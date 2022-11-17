package minermock

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

type StratumV1ResultHandler = func(a stratumv1_message.MiningResult)

const NEWLINE = '\n'

type MinerMock struct {
	dest interfaces.IDestination
	id   atomic.Uint64 // message counter

	conn        net.Conn
	resHandlers sync.Map // map of ID to handler functions
	diff        float64  // current difficulty

	log interfaces.ILogger
}

func NewMinerMock(dest interfaces.IDestination) *MinerMock {
	id := atomic.Uint64{}
	id.Store(0)

	return &MinerMock{
		dest: dest,
		id:   id,
		log:  lib.NewTestLogger(),
	}
}

func (m *MinerMock) Run() error {
	conn, err := net.Dial("tcp", m.dest.GetHost())
	if err != nil {
		return err
	}

	m.conn = conn

	ctx := context.Background()
	errCh := make(chan error)
	doneCh := make(chan struct{})

	handshakeDone := make(chan struct{})

	go func() {
		err := m.readMessages(ctx)
		if err != nil {
			errCh <- err
		}
		close(doneCh)
	}()

	go func() {
		err := m.handshake(ctx)
		if err != nil {
			errCh <- err
		}
		close(handshakeDone)
	}()

	go func() {
		<-handshakeDone
		<-time.After(5 * time.Second)

		for {
			msg := stratumv1_message.NewMiningSubmit(m.dest.Username(), "620daf25f", "0000000000000000", "62cea7a6", "f9b40000")
			err := m.sendAndAwait(context.Background(), msg)
			if err != nil {
				m.log.Warnf("submit error %s", err)
			}
			delay := rand.Float32() * 5 * float32(time.Second)
			<-time.After(time.Duration(delay))
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-doneCh:
		return nil
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

func (m *MinerMock) sendAndAwait(ctx context.Context, msg stratumv1_message.MiningMessageToPool) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	ID := m.acquireID()
	msg.SetID(ID)

	bytes := append(msg.Serialize(), byte(NEWLINE))
	_, err := m.conn.Write(bytes)
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	_, err = m.awaitResponse(ID)
	if err != nil {
		return err
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
			m.log.Infof("received Notify")

		case *stratumv1_message.MiningSetDifficulty:
			m.diff = typedMessage.GetDifficulty()
			m.log.Infof("received SetDifficulty %.2f", m.diff)

		case *stratumv1_message.MiningResult:
			id := typedMessage.GetID()
			m.log.Infof("received Result %d", id)
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
	case <-time.After(30 * time.Second):
		return stratumv1_message.MiningResult{}, fmt.Errorf("pool response timeout")
	}
}
