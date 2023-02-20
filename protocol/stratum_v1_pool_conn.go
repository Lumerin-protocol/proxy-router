package protocol

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gammazero/deque"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
	"go.uber.org/atomic"
)

const ReadBufferSize = 100 // number of messages in read buffer

var ErrReadBufferFull = errors.New("pool connection read buffer is full")

// StratumV1PoolConn represents connection to the pool on the protocol level
type StratumV1PoolConn struct {
	// dependencies
	conn net.Conn
	log  interfaces.ILogger

	// configuration
	dest        interfaces.IDestination
	connTimeout time.Duration
	logStratum  bool

	// internal state
	readBuffer chan stratumv1_message.MiningMessageGeneric // auxillary channel to relay messages

	notifyMsgs    *deque.Deque[*stratumv1_message.MiningNotify] // recent relevant notify messages, (respects stratum clean_jobs flag)
	setDiffMsg    *stratumv1_message.MiningSetDifficulty        // recent difficulty message
	extraNonceMsg *stratumv1_message.MiningSetExtranonce        // keeps relevant extranonce (from mining.subscribe response) TODO: handle pool setExtranonce message
	configureMsg  *stratumv1_message.MiningConfigure

	isReading     atomic.Bool   // if false messages will not be availabe to read from outside, used for authentication handshake
	isConnectedCh chan struct{} // was connect called before read or write

	lastRequestId *atomic.Uint32 // stratum request id counter
	resHandlers   sync.Map       // allows to register callbacks for particular messages to simplify transaction flow

	newCloseTimeout      chan time.Time // channel with newly set close timeouts, nil means timeout not set or already expired
	newCloseTimeoutMutex sync.Mutex     // guards newCloseTimeout
	closeTimeout         time.Time      // last set timeout
}

func NewStratumV1Pool(conn net.Conn, log interfaces.ILogger, dest interfaces.IDestination, configureMsg *stratumv1_message.MiningConfigure, connTimeout time.Duration, logStratum bool) *StratumV1PoolConn {
	return &StratumV1PoolConn{
		dest: dest,

		conn:         conn,
		notifyMsgs:   deque.New[*stratumv1_message.MiningNotify](20, 20),
		configureMsg: configureMsg,

		readBuffer: make(chan stratumv1_message.MiningMessageGeneric, ReadBufferSize),

		isReading:     *atomic.NewBool(false),
		isConnectedCh: make(chan struct{}),

		lastRequestId: atomic.NewUint32(0),
		resHandlers:   sync.Map{},

		connTimeout: connTimeout,

		log:        log,
		logStratum: logStratum,
	}
}

// Run enables proxying and handling pool messages
func (c *StratumV1PoolConn) Run(ctx context.Context) error {
	c.readBuffer = make(chan stratumv1_message.MiningMessageGeneric, ReadBufferSize)

	defer func() {
		// cleanup
		close(c.readBuffer)
		c.resHandlers = sync.Map{}
		c.notifyMsgs = nil
	}()

	sourceReader := bufio.NewReader(c.conn)
	c.log.Debug("pool reader started")

	for {
		line, isPrefix, err := sourceReader.ReadLine()
		if isPrefix {
			return fmt.Errorf("line is too long")
		}

		if err != nil {
			return err
		}

		if c.logStratum {
			lib.LogMsg(false, true, c.dest.GetHost(), line, c.log)
		}

		m, err := stratumv1_message.ParseMessageFromPool(line)
		if err != nil {
			c.log.Errorf("unknown pool message", string(line))
		}

		m = c.readInterceptor(m)

		if c.isReading.Load() {
			select {
			case c.readBuffer <- m:
			default:
				return ErrReadBufferFull
			}
		}
	}
}

// Connect initiates connection handshake. Make sure m.Run was called
func (c *StratumV1PoolConn) Connect(ctx context.Context) error {
	c.log.Debug("start connecting")

	if c.configureMsg != nil {
		_, err := c.SendPoolRequestWait(ctx, c.configureMsg)
		if err != nil {
			return err
		}
	}

	subscribeRes, err := c.SendPoolRequestWait(ctx, stratumv1_message.NewMiningSubscribe(1, "miner", ""))
	if err != nil {
		return err
	}
	if subscribeRes.IsError() {
		return fmt.Errorf("invalid subscribe response %s", subscribeRes.Serialize())
	}
	c.log.Debug("connect: subscribe sent")

	extranonce, extranonceSize, err := stratumv1_message.ParseExtranonceSubscribeResult(subscribeRes)
	if err != nil {
		return err
	}

	c.extraNonceMsg = stratumv1_message.NewMiningSetExtranonceV2(extranonce, extranonceSize)

	authMsg := stratumv1_message.NewMiningAuthorize(1, c.dest.Username(), c.dest.Password())
	_, err = c.SendPoolRequestWait(ctx, authMsg)
	if err != nil {
		c.log.Debugf("reconnect: error sent subscribe %w", err)
		return err
	}
	c.log.Debug("connect: authorize sent")

	close(c.isConnectedCh)

	return nil
}

// SendPoolRequestWait sends a message and awaits for the response
func (c *StratumV1PoolConn) SendPoolRequestWait(ctx context.Context, msg stratumv1_message.MiningMessageToPool) (*stratumv1_message.MiningResult, error) {
	c.log.Debug("sending message... %s", msg.Serialize())
	id := int(c.lastRequestId.Inc())
	msg.SetID(id)

	err := c.write(ctx, msg)
	if err != nil {
		return nil, err
	}
	errCh := make(chan error)
	resCh := make(chan stratumv1_message.MiningResult)

	c.RegisterResultHandler(id, func(a stratumv1_message.MiningResult) stratumv1_message.MiningMessageGeneric {
		if a.IsError() {
			errCh <- errors.New(a.GetError())
		} else {
			resCh <- a
		}
		return nil // do not proxy this request
	})

	select {
	case err := <-errCh:
		return nil, err
	case res := <-resCh:
		return &res, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *StratumV1PoolConn) RegisterResultHandler(id int, handler StratumV1ResultHandler) {
	c.resHandlers.Store(fmt.Sprint(id), handler)
}

// Pauses emitting any pool messages, then sends cached messages for a recent job, and then resumes pool message flow
func (c *StratumV1PoolConn) ResendRelevantNotifications(ctx context.Context) error {
	c.Release()
	err := c.sendToReadCh(ctx, c.extraNonceMsg)
	if err != nil {
		return err
	}
	c.log.Debugf("extranonce was resent")

	if c.setDiffMsg != nil {
		err = c.sendToReadCh(ctx, c.setDiffMsg)
		if err != nil {
			return err
		}
		c.log.Debugf("set-difficulty was resent")
	}

	for {
		if c.notifyMsgs.Len() == 0 {
			break
		}
		msg := c.notifyMsgs.PopBack()
		err = c.sendToReadCh(ctx, msg)
		if err != nil {
			return err
		}
	}
	c.log.Debugf("notify msgs were resent")

	return nil
}

func (c *StratumV1PoolConn) sendToReadCh(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error {
	select {
	case c.readBuffer <- msg:
		return nil
	default:
		return ErrReadBufferFull
	}
}

// Read reads message from pool
func (c *StratumV1PoolConn) Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error) {
	<-c.isConnectedCh // wait until connection established
	c.isReading.Store(true)

	select {
	case msg := <-c.readBuffer:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *StratumV1PoolConn) Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error {
	<-c.isConnectedCh // wait until connection established
	return c.write(ctx, msg)
}

// Write writes message to pool
func (c *StratumV1PoolConn) write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error {
	msg = c.writeInterceptor(msg)

	if c.logStratum {
		if c.dest != nil && msg != nil {
			lib.LogMsg(false, false, c.dest.GetHost(), msg.Serialize(), c.log)
		}
	}

	b := append(msg.Serialize(), lib.CharNewLine)
	_, err := c.conn.Write(b)

	if err != nil {
		return err
	}

	return nil
}

// Returns current extranonce values
func (c *StratumV1PoolConn) GetExtranonce() (string, int) {
	return c.extraNonceMsg.GetExtranonce()
}

func (c *StratumV1PoolConn) GetDest() interfaces.IDestination {
	return c.dest
}

func (c *StratumV1PoolConn) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

// readInterceptor caches relevant messages and invokes callbacks
func (c *StratumV1PoolConn) readInterceptor(m stratumv1_message.MiningMessageGeneric) stratumv1_message.MiningMessageGeneric {
	switch typedMessage := m.(type) {
	case *stratumv1_message.MiningNotify:
		if typedMessage.GetCleanJobs() {
			c.notifyMsgs.Clear()
		}
		c.notifyMsgs.PushBack(typedMessage.Copy())

	case *stratumv1_message.MiningSetDifficulty:
		c.setDiffMsg = typedMessage.Copy()

	case *stratumv1_message.MiningResult:
		id := typedMessage.GetID()
		handler, ok := c.resHandlers.LoadAndDelete(fmt.Sprint(id))
		if ok {
			handledMsg := handler.(StratumV1ResultHandler)(*typedMessage.Copy())
			if handledMsg != nil {
				m = handledMsg.(*stratumv1_message.MiningResult)
			}
		}
	}

	return m
}

func (c *StratumV1PoolConn) writeInterceptor(m stratumv1_message.MiningMessageGeneric) stratumv1_message.MiningMessageGeneric {
	switch typedMsg := m.(type) {
	case *stratumv1_message.MiningSubmit:
		typedMsg.SetWorkerName(c.dest.Username())
		m = typedMsg
	}
	return m
}

// Release stops buffering messages, only collects notify and set difficulty, should be called before changing destination
func (c *StratumV1PoolConn) Release() {
	c.isReading.Store(false)
	c.readBuffer = make(chan stratumv1_message.MiningMessageGeneric, 100)
}

func (c *StratumV1PoolConn) Close() error {
	err := c.conn.Close()
	c.resHandlers = sync.Map{}
	c.notifyMsgs = nil
	return err
}

func (c *StratumV1PoolConn) CloseTimeout(cleanupCb func()) {
	c.newCloseTimeout = make(chan time.Time)

	go func() {
		var closeTimeoutChan <-chan time.Time

		for {
			select {
			case <-closeTimeoutChan:
				cleanupCb()

				c.newCloseTimeoutMutex.Lock()
				newCloseTimeoutRef := c.newCloseTimeout
				c.newCloseTimeout = nil
				c.newCloseTimeoutMutex.Unlock()

				close(newCloseTimeoutRef)
				err := c.Close()
				if err != nil {
					c.log.Errorf("close connection after timeout error %s", err)
				}

				return
			case timeout := <-c.newCloseTimeout:
				if timeout.IsZero() {
					return
				}
				duration := time.Until(timeout)
				c.closeTimeout = timeout
				closeTimeoutChan = time.After(duration)
			}
		}

	}()
}

func (c *StratumV1PoolConn) SetCloseTimeout(timeout time.Time) {
	c.newCloseTimeoutMutex.Lock()
	defer c.newCloseTimeoutMutex.Unlock()

	if c.newCloseTimeout != nil {
		c.newCloseTimeout <- timeout
	}
}

func (c *StratumV1PoolConn) GetCloseTimeout() time.Time {
	return c.closeTimeout
}
