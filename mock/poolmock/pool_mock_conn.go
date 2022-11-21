package poolmock

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

const (
	INIT_DIFF        = 8096.0
	NOTIFY_INTERVAL  = 30 * time.Second
	RESPONSE_TIMEOUT = 30 * time.Second
)

type StratumV1MsgHandler = func(a stratumv1_message.MiningMessageToPool)

type PoolMockConn struct {
	conn        net.Conn
	msgHandlers sync.Map
	diff        float64
	diffCh      chan float64
	workerName  string
	id          string
	submitCount atomic.Int64

	log interfaces.ILogger
}

func NewPoolMockConn(conn net.Conn, log interfaces.ILogger) *PoolMockConn {
	return &PoolMockConn{
		conn: conn,
		log:  log,
		diff: INIT_DIFF,
		id:   conn.LocalAddr().String(),
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

func (c *PoolMockConn) Run(ctx context.Context) error {
	errCh := make(chan error)
	doneCh := make(chan struct{})

	handshakeDone := make(chan struct{})

	go func() {
		err := c.readMessages(ctx)
		if err != nil {
			errCh <- err
		}
		close(doneCh)
	}()

	go func() {
		err := c.handshake(ctx)
		if err != nil {
			errCh <- err
		}
		close(handshakeDone)
	}()

	go func() {
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		case <-handshakeDone:
		}

		go func() {
			err := c.watchDiff(ctx)
			if err != nil {
				errCh <- err
			}
			close(doneCh)
		}()

		go func() {
			err := c.watchNotify(ctx)
			if err != nil {
				errCh <- err
			}
			close(doneCh)
		}()

	}()

	select {
	case err := <-errCh:
		return err
	case <-doneCh:
		return nil
	}
}

func (c *PoolMockConn) watchNotify(ctx context.Context) error {
	for {
		err := c.notify(ctx)
		if err != nil {
			return err
		}

		select {
		case <-time.After(NOTIFY_INTERVAL):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *PoolMockConn) watchDiff(ctx context.Context) error {
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

func (c *PoolMockConn) handshake(ctx context.Context) error {
	subscribeReceivedCh := make(chan any)
	authorizeReceivedCh := make(chan any)
	errCh := make(chan error)

	c.registerHandlerTimeout(ctx, stratumv1_message.MethodMiningSubscribe, func(msg stratumv1_message.MiningMessageToPool) {
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

	c.registerHandlerTimeout(ctx, stratumv1_message.MethodMiningAuthorize, func(msg stratumv1_message.MiningMessageToPool) {
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

	c.registerHandlerTimeout(ctx, stratumv1_message.MethodMiningConfigure, func(msg stratumv1_message.MiningMessageToPool) {
		err := c.send(ctx, &stratumv1_message.MiningResult{
			ID:     msg.GetID(),
			Result: []byte(`{"version-rolling":true,"version-rolling.mask":"1fffe000"}`),
		})
		if err != nil {
			c.log.Errorf("configure msg error %s", err)
		}
	})

	select {
	case <-waitCh(subscribeReceivedCh, authorizeReceivedCh):
		return nil
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *PoolMockConn) notify(ctx context.Context) error {
	msg, _ := stratumv1_message.ParseMiningNotify([]byte(`{"id":null,"method":"mining.notify","params":["620e41a18","b56266ef4c94ba61562510b7656d132cacc928c50008488c0000000000000000","01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff4b03795d0bfabe6d6d021f1a6edc2237ed5d6b5ce13c9b8516dcae143a5cf8a373fa0406c7ae06fb260100000000000000","181ae420062f736c7573682f00000000030f1b0a27000000001976a9147c154ed1dc59609e3d26abb2df2ea3d587cd8c4188ac00000000000000002c6a4c2952534b424c4f434b3ad6b225f8545c851f458a3f603c9a5bd63959ab646f977ff2fd8d1e2f004427dc0000000000000000266a24aa21a9ed46c37118ea57f6c2152b2f156e4df28edb997e21ea8fe5754e9a869f0287cfb800000000",["bed26bac18890c62ab48bdd913ab2b648326286607b3a159987cf36b6fe55d7e","8c39bd3ac4aeedc7b3c354008692ccb5e758a9cd9b1888a72090268365d9fbf0","00dca1b0b193f0298f155d32a9c04a79a49a2617853b787a79adc942cae74fed","fbb3b3a6bf5710f885fd377a2fde24fbb795933c9e6ceea67f91f1d90c532be2","ddd51d322b9c61621f762002dc179de1c24f4454e17943de172e24e9ad4be942","6fcffd6b0ebd01f15f57cb6ab3fabe151d757f1f71b45ef420b97b5fac0cc670","c28b6ef7c87ff5982bec0eaccb1855fff78397b2c732030dc792636a9492ba7c","afe4f418f45c78c36a848930b56323749fb7da453d557aa41c820a3170f7d20a","c61bab8479a8ada4c9761be0ce82e3183e7405b57b31925e0e411e73376fb22e","224b536c03f1379f708172970fca51160192bd8eded9cf578f7f5c3c8795eb33","fe4af4678dd3f66946738a1479627683e461bb832629a01978b4f2165490f460"],"20000004","1709a7af","62cea7e2",false]}`))
	return c.send(ctx, msg)
}

func (c *PoolMockConn) setDifficulty(ctx context.Context, diff float64) error {
	msg := stratumv1_message.NewMiningSetDifficulty(diff)
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
			c.log.Infof("received subscribe")
			handler, ok := c.msgHandlers.LoadAndDelete(stratumv1_message.MethodMiningSubscribe)
			if ok {
				handler.(StratumV1MsgHandler)(typedMessage)
			}

		case *stratumv1_message.MiningAuthorize:
			c.log.Infof("received authorize")
			handler, ok := c.msgHandlers.LoadAndDelete(stratumv1_message.MethodMiningAuthorize)
			if ok {
				handler.(StratumV1MsgHandler)(typedMessage)
			}

		case *stratumv1_message.MiningConfigure:
			c.log.Infof("received configure")
			handler, ok := c.msgHandlers.LoadAndDelete(stratumv1_message.MethodMiningConfigure)
			if ok {
				handler.(StratumV1MsgHandler)(typedMessage)
			}

		case *stratumv1_message.MiningSubmit:
			c.submitCount.Add(1)
			c.log.Infof("received submit")
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

func (c *PoolMockConn) registerHandlerTimeout(ctx context.Context, method string, f StratumV1MsgHandler) error {
	ctx, cancel := context.WithTimeout(ctx, RESPONSE_TIMEOUT)
	defer cancel()

	msgCh := make(chan stratumv1_message.MiningMessageToPool)

	c.registerHandler(method, func(msg stratumv1_message.MiningMessageToPool) {
		select {
		case msgCh <- msg:
		case <-ctx.Done():
		}
		close(msgCh)
	})

	select {
	case msg := <-msgCh:
		f(msg)
		return nil
	case <-ctx.Done():
		return ctx.Err()
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

	c.log.Infof("sent msg %s", string(msg.Serialize()))
	return nil
}
