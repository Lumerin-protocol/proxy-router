package poolmock

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gammazero/deque"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

const (
	NOTIFY_INTERVAL        = 120 * time.Second // interval between new notify messages
	HANDSHAKE_TIMEOUT      = 30 * time.Second  // if handshake takes more than specified - pool errors
	VAR_DIFF_TIME          = 60 * time.Second  // if no submits during this duration difficulty will decrease
	VAR_DIFF_AVERAGE_COUNT = 5                 // if average time of N last submits less than VAR_DIFF_TIME/2 difficulty will increase
)

type StratumV1MsgHandler = func(a stratumv1_message.MiningMessageToPool)
type OnSubmitHandler = func(workerName string, msgID int, diff int64)

type PoolMockConn struct {
	conn        net.Conn
	msgHandlers sync.Map
	workerName  string
	id          string

	submitCount atomic.Int64
	onSubmit    OnSubmitHandler

	diff                 int
	diffCh               chan int
	varDiffCount         *VarDiff
	isVarDiff            atomic.Bool
	varDiffSubmitCh      chan int
	varDiffSubmitCounter *deque.Deque[time.Time]

	log interfaces.ILogger
}

func NewPoolMockConn(conn net.Conn, varDiffRange [2]int, onSubmit OnSubmitHandler, log interfaces.ILogger) *PoolMockConn {
	isVarDiff := atomic.Bool{}
	isVarDiff.Store(true)

	return &PoolMockConn{
		conn:                 conn,
		log:                  log,
		diff:                 varDiffRange[0],
		diffCh:               make(chan int),
		isVarDiff:            isVarDiff,
		onSubmit:             onSubmit,
		id:                   conn.RemoteAddr().String(),
		varDiffCount:         NewVarDiff(varDiffRange),
		varDiffSubmitCounter: deque.New[time.Time](8, 8),
		varDiffSubmitCh:      make(chan int, 5),
	}
}

func (c *PoolMockConn) ID() string {
	return c.id
}

func (c *PoolMockConn) GetWorkerName() string {
	return c.workerName
}

func (c *PoolMockConn) GetSubmitCount() int {
	return int(c.submitCount.Load())
}

func (c *PoolMockConn) SetDifficulty(diff int) {
	c.isVarDiff.Store(false)
	c.log.Infof("new difficulty: %d", diff)
	c.diffCh <- diff
}

func (c *PoolMockConn) EnableVarDiff() {
	c.isVarDiff.Store(true)
}

func (c *PoolMockConn) Run(ctx context.Context) error {
	errCh := make(chan error, 5)

	handshakeCtx, cancel := context.WithTimeout(ctx, HANDSHAKE_TIMEOUT)
	defer cancel()

	waitHandshake := c.handshake(handshakeCtx)

	go func() {
		errCh <- c.readMessages(ctx)
	}()

	err := waitHandshake()
	if err != nil {
		errCh <- err
	}

	c.log.Infof("handshake completed")

	go func() {
		errCh <- c.watchDiff(ctx)
	}()

	go func() {
		errCh <- c.varDiff(ctx)
	}()

	go func() {
		errCh <- c.watchNotify(ctx)
	}()

	return <-errCh
}

func (c *PoolMockConn) watchNotify(ctx context.Context) error {
	for {
		err := c.notify(ctx)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(NOTIFY_INTERVAL):
		}
	}
}

func (c *PoolMockConn) watchDiff(ctx context.Context) error {
	err := c.setDifficulty(ctx, c.diff)
	if err != nil {
		return err
	}

	for {
		select {
		case diff := <-c.diffCh:
			c.diff = diff
			err := c.setDifficulty(ctx, c.diff)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *PoolMockConn) varDiff(ctx context.Context) error {
	for {
		// if pool gets submit between VAR_DIFF_TIME/2 and VAR_DIFF_TIME - noop
		// if less than VAR_DIFF_TIME/2 - increase diff
		// if more than VAR_DIFF_TIME - decrease diff
		tillDiffDecrease := VAR_DIFF_TIME
		select {
		case <-time.After(tillDiffDecrease):
			// decrease vardiff
			c.log.Debugf("decreasing diff: VAR_DIFF_TIME elapsed (%s)", VAR_DIFF_TIME)
			ok := c.varDiffCount.Dec()
			if ok {
				newDiff := c.varDiffCount.Val()
				c.SetDifficulty(newDiff)
				c.varDiffSubmitCounter.Clear()
			}
		case <-c.varDiffSubmitCh:
			// get average time over 5 last submits
			// increase vardiff
			if c.varDiffSubmitCounter.Len() >= VAR_DIFF_AVERAGE_COUNT {
				oldest := c.varDiffSubmitCounter.At(0)
				elapsed := time.Since(oldest)
				timePerSubmit := elapsed / time.Duration(VAR_DIFF_AVERAGE_COUNT)

				if timePerSubmit < VAR_DIFF_TIME/2 {
					c.log.Debugf("increasing diff: average submit time for (%d) recent submits is (%s) which is less than VAR_DIFF_TIME/2 (%s)",
						VAR_DIFF_AVERAGE_COUNT, timePerSubmit, VAR_DIFF_TIME/2)

					ok := c.varDiffCount.Inc()
					if ok {
						newDiff := c.varDiffCount.Val()
						c.SetDifficulty(newDiff)
						c.varDiffSubmitCounter.Clear()
					}
				}
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *PoolMockConn) handshake(ctx context.Context) func() error {
	subscribeReceivedCh := make(chan any)
	authorizeReceivedCh := make(chan any)
	errCh := make(chan error, 3)

	c.registerHandler(stratumv1_message.MethodMiningSubscribe, func(msg stratumv1_message.MiningMessageToPool) {
		err := c.send(ctx, &stratumv1_message.MiningResult{
			ID:     msg.GetID(),
			Result: []byte(`[[["mining.set_difficulty","1"],["mining.notify","1"]],"086502001b720a",8]`),
		})
		if err != nil {
			c.log.Errorf("configure msg error %s", err)
			errCh <- err
			return
		}
		close(subscribeReceivedCh)
	})

	c.registerHandler(stratumv1_message.MethodMiningAuthorize, func(msg stratumv1_message.MiningMessageToPool) {
		typedMsg := msg.(*stratumv1_message.MiningAuthorize)
		c.workerName = typedMsg.GetWorkerName()
		err := c.send(ctx, &stratumv1_message.MiningResult{
			ID:     msg.GetID(),
			Result: []byte("true"),
		})
		if err != nil {
			c.log.Errorf("configure msg error %s", err)
			errCh <- err
			return
		}
		close(authorizeReceivedCh)
	})

	c.registerHandler(stratumv1_message.MethodMiningConfigure, func(msg stratumv1_message.MiningMessageToPool) {
		err := c.send(ctx, &stratumv1_message.MiningResult{
			ID:     msg.GetID(),
			Result: []byte(`{"version-rolling":true,"version-rolling.mask":"1fffe000"}`),
		})
		if err != nil {
			c.log.Errorf("configure msg error %s", err)
		}
	})

	return func() error {
		select {
		case <-waitCh(subscribeReceivedCh, authorizeReceivedCh):
			return nil
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *PoolMockConn) notify(ctx context.Context) error {
	// creating different notify messages to simulate load for real miners (for example cpuminer)
	jobIDHex := strconv.FormatInt(int64(rand.Int31()), 16)

	// random 128 bit hex
	h := sha256.New()
	_, err := h.Write([]byte(jobIDHex))
	if err != nil {
		return err
	}
	rand128bitHex := fmt.Sprintf("%x", h.Sum(nil))

	coinbase1 := fmt.Sprintf("%s%s%s", COINBASE1_PREFIX, rand128bitHex, COINBASE1_SUFFIX)
	msg, _ := stratumv1_message.ParseMiningNotify([]byte(NOTIFY_MSG))

	msg.SetJobID(jobIDHex)
	msg.SetGen1(coinbase1)

	defer c.log.Infof("notify sent")

	return c.send(ctx, msg)
}

func (c *PoolMockConn) setDifficulty(ctx context.Context, diff int) error {
	msg := stratumv1_message.NewMiningSetDifficulty(float64(diff))
	return c.send(ctx, msg)
}

func (c *PoolMockConn) readMessages(ctx context.Context) error {
	sourceReader := bufio.NewReader(c.conn)
	for {
		line, isPrefix, err := sourceReader.ReadLine()
		if isPrefix {
			return fmt.Errorf("line is too long")
		}
		if err != nil {
			return err
		}

		msg, err := stratumv1_message.ParseMessageToPool(line, c.log)
		if err != nil {
			return err
		}

		switch typedMessage := msg.(type) {
		case *stratumv1_message.MiningSubscribe:
			c.log.Debugf("received subscribe")
			handler, ok := c.msgHandlers.LoadAndDelete(stratumv1_message.MethodMiningSubscribe)
			if ok {
				handler.(StratumV1MsgHandler)(typedMessage)
			}

		case *stratumv1_message.MiningAuthorize:
			c.log.Debugf("received authorize")
			handler, ok := c.msgHandlers.LoadAndDelete(stratumv1_message.MethodMiningAuthorize)
			if ok {
				handler.(StratumV1MsgHandler)(typedMessage)
			}

		case *stratumv1_message.MiningConfigure:
			c.log.Debugf("received configure")
			handler, ok := c.msgHandlers.LoadAndDelete(stratumv1_message.MethodMiningConfigure)
			if ok {
				handler.(StratumV1MsgHandler)(typedMessage)
			}

		case *stratumv1_message.MiningSubmit:
			c.submitCount.Add(1)
			c.onSubmit(c.workerName, msg.GetID(), int64(c.diff))
			c.varDiff2(c.diff)

			handler, ok := c.msgHandlers.LoadAndDelete(stratumv1_message.MethodMiningSubmit)
			if ok {
				handler.(StratumV1MsgHandler)(typedMessage)
			}
			err := c.send(ctx, &stratumv1_message.MiningResult{
				ID:     msg.GetID(),
				Result: []byte(`true`),
			})
			if err != nil {
				return err
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

func (c *PoolMockConn) registerHandler(method string, f StratumV1MsgHandler) {
	c.msgHandlers.Store(method, f)
}

func (c *PoolMockConn) send(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error {
	bytes := append(msg.Serialize(), lib.CharNewLine)
	_, err := c.conn.Write(bytes)
	if err != nil {
		return err
	}

	c.log.Debugf("sent msg %s", string(msg.Serialize()))
	return nil
}

func (c *PoolMockConn) varDiff2(diff int) {
	if c.varDiffSubmitCounter.Len() >= VAR_DIFF_AVERAGE_COUNT {
		_ = c.varDiffSubmitCounter.PopFront()
	}
	c.varDiffSubmitCounter.PushBack(time.Now())
	c.varDiffSubmitCh <- diff
}
