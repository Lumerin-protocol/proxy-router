package minermock

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

type StratumV1ResultHandler = func(a stratumv1_message.MiningResult)

const (
	INIT_TIMEOUT            = 5 * time.Second
	DEFAULT_SUBMIT_INTERVAL = 5 * time.Second
	RESPONSE_TIMEOUT        = time.Minute
)

type MinerMock struct {
	dest interfaces.IDestination
	id   atomic.Uint64 // message counter

	conn           net.Conn
	resHandlers    sync.Map // map of ID to handler functions
	diff           float64  // current difficulty
	submitInterval time.Duration

	log interfaces.ILogger
}

func NewMinerMock(dest interfaces.IDestination, log interfaces.ILogger) *MinerMock {
	id := atomic.Uint64{}
	id.Store(0)

	return &MinerMock{
		dest:           dest,
		id:             id,
		log:            log,
		submitInterval: DEFAULT_SUBMIT_INTERVAL,
	}
}

func (m *MinerMock) GetWorkerName() string {
	return m.dest.Username()
}

func (m *MinerMock) SetSubmitInterval(interval time.Duration) {
	m.submitInterval = interval
}

func (m *MinerMock) Run(ctx context.Context) error {
	conn, err := net.Dial("tcp", m.dest.GetHost())
	if err != nil {
		return err
	}

	m.log.Info("miner connected to pool")

	m.conn = conn

	errCh := make(chan error)
	doneCh := make(chan struct{})
	handshakeDone := make(chan struct{})

	go func() {
		err := m.readMessages(ctx)
		if err != nil {
			errCh <- err
			return
		}
		close(doneCh)
	}()

	go func() {
		err := m.handshake(ctx)
		if err != nil {
			errCh <- err
			return
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

		m.log.Info("handshake completed")

		err := m.mine(ctx)
		if err != nil {
			errCh <- err
		}
		close(doneCh)
	}()

	select {
	case err := <-errCh:
		m.log.Warnf("miner error %s", err)
		return err
	case <-doneCh:
		m.log.Info("miner exited")
		return nil
	}
}

func (m *MinerMock) mine(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.submitInterval):
		}

		err := m.submit(ctx)
		if err != nil {
			return err
		}
	}
}

func (m *MinerMock) handshake(ctx context.Context) error {
	err := m.subscribe(ctx)
	if err != nil {
		return err
	}

	err = m.authorize(ctx)
	if err != nil {
		return err
	}

	err = m.configure(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (m *MinerMock) subscribe(ctx context.Context) error {
	msg := stratumv1_message.NewMiningSubscribe(0, "miner", "")
	return m.sendAndAwait(ctx, msg)
}

func (m *MinerMock) authorize(ctx context.Context) error {
	msg := stratumv1_message.NewMiningAuthorize(0, m.dest.Username(), m.dest.Password())
	return m.sendAndAwait(ctx, msg)
}

func (m *MinerMock) configure(ctx context.Context) error {
	msg := stratumv1_message.NewMiningConfigure(
		[]string{"minimum-difficulty", "version-rolling"},
		map[string]any{
			"minimum-difficulty.value":      2048,
			"version-rolling.mask":          "1fffe000",
			"version-rolling.min-bit-count": 2,
		},
	)
	return m.sendAndAwait(ctx, msg)
}

func (m *MinerMock) submit(ctx context.Context) error {
	msg := stratumv1_message.NewMiningSubmit(m.dest.Username(), "620daf25f", "0000000000000000", "62cea7a6", "f9b40000")
	m.log.Info("new submit")
	return m.sendAndAwait(ctx, msg)
}

func (m *MinerMock) sendAndAwait(ctx context.Context, msg stratumv1_message.MiningMessageToPool) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	ID := m.acquireID()
	msg.SetID(ID)

	errCh := make(chan error)

	go func() {
		_, err := m.awaitResponse(ID)
		errCh <- err
		close(errCh)
	}()

	bytes := append(msg.Serialize(), lib.CharNewLine)
	_, err := m.conn.Write(bytes)
	if err != nil {
		return err
	}

	m.log.Debugf("sent msg %d %s", ID, string(msg.Serialize()))

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (m *MinerMock) acquireID() int {
	return int(m.id.Add(1))
}

func (m *MinerMock) readMessages(ctx context.Context) error {
	sourceReader := bufio.NewReader(m.conn)
	for {
		line, isPrefix, err := sourceReader.ReadLine()
		if isPrefix {
			return fmt.Errorf("line is too long")
		}
		if err != nil {
			return err
		}

		msg, err := stratumv1_message.ParseMessageFromPool(line)
		if err != nil {
			return err
		}

		switch typedMessage := msg.(type) {
		case *stratumv1_message.MiningNotify:
			m.log.Debugf("received Notify")

		case *stratumv1_message.MiningSetDifficulty:
			m.diff = typedMessage.GetDifficulty()
			m.log.Debugf("received SetDifficulty %.2f", m.diff)

		case *stratumv1_message.MiningResult:
			id := typedMessage.GetID()
			m.log.Debugf("received Result %d %s %s", id, typedMessage.Result, typedMessage.GetError())
			handler, ok := m.resHandlers.LoadAndDelete(id)
			if ok {
				handler.(StratumV1ResultHandler)(*typedMessage)
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

func (m *MinerMock) registerHandler(ID int, f StratumV1ResultHandler) {
	m.resHandlers.Store(ID, f)
}

func (m *MinerMock) awaitResponse(ID int) (stratumv1_message.MiningResult, error) {
	msgCh := make(chan stratumv1_message.MiningResult)

	m.registerHandler(ID, func(a stratumv1_message.MiningResult) {
		msgCh <- a
		close(msgCh)
	})

	select {
	case msg := <-msgCh:
		return msg, nil
	case <-time.After(RESPONSE_TIMEOUT):
		return stratumv1_message.MiningResult{}, fmt.Errorf("pool response timeout")
	}
}
