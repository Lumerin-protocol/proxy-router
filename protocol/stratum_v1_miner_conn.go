package protocol

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

type StratumV1Miner struct {
	conn        net.Conn
	reader      *bufio.Reader
	connectedAt time.Time
	connTimeout time.Duration

	isWriting bool        // used to temporarily pause writing messages to miner
	mu        *sync.Mutex // guards isWriting

	cond          *sync.Cond
	extraNonceMsg *stratumv1_message.MiningSubscribeResult
	workerName    string
	log           interfaces.ILogger
	logStratum    bool
}

func NewStratumV1MinerConn(conn net.Conn, log interfaces.ILogger, extraNonce *stratumv1_message.MiningSubscribeResult, logStratum bool, connectedAt time.Time, connTimeout time.Duration) *StratumV1Miner {
	mu := new(sync.Mutex)
	return &StratumV1Miner{
		conn:          conn,
		reader:        bufio.NewReader(conn),
		connectedAt:   connectedAt,
		isWriting:     false,
		mu:            mu,
		cond:          sync.NewCond(mu),
		extraNonceMsg: extraNonce,
		log:           log,
		logStratum:    logStratum,
		connTimeout:   connTimeout,
	}
}

func (m *StratumV1Miner) Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error {
	return m.write(ctx, msg)
}

// write writes to miner omitting locks
func (m *StratumV1Miner) write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error {
	if m.conn != nil && msg != nil {
		if m.logStratum {
			lib.LogMsg(true, false, m.conn.RemoteAddr().String(), msg.Serialize(), m.log)
		}

		b := append(msg.Serialize(), lib.CharNewLine)
		_, err := m.conn.Write(b)
		return err
	}

	return fmt.Errorf("invalid message or connection; connection: %v; message: %v", m.conn, msg)
}

func (s *StratumV1Miner) Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error) {
	doneCh := make(chan struct{})
	defer close(doneCh)

	// cancellation via context is implemented using SetReadDeadline,
	// which unblocks read operation causing it to return os.ErrDeadlineExceeded
	// TODO: consider implementing it in separate goroutine instead of a goroutine per read
	go func() {
		select {
		case <-ctx.Done():
			// may return ErrClosing
			err := s.conn.SetReadDeadline(time.Now())
			if err != nil {
				s.log.Warnf("err during setting SetReadDeadline to unblock reading: %s", err)
			}
		case <-doneCh:
			return
		}
	}()

	for {
		err := s.conn.SetReadDeadline(time.Now().Add(s.connTimeout))
		if err != nil {
			return nil, err
		}

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		line, isPrefix, err := s.reader.ReadLine()

		if isPrefix {
			return nil, fmt.Errorf("line is too long")
		}

		if err != nil {
			// return correct error if read was cancelled via context
			if ctx.Err() != nil && errors.Is(err, os.ErrDeadlineExceeded) {
				return nil, ctx.Err()
			}
			return nil, err
		}

		if s.logStratum {
			lib.LogMsg(true, true, s.conn.RemoteAddr().String(), line, s.log)
		}

		m, err := stratumv1_message.ParseMessageToPool(line, s.log)

		if err != nil {
			s.log.Errorf("unknown miner message: %s", string(line))
			continue
		}

		return m, nil
	}
}

func (s *StratumV1Miner) GetID() string {
	return s.conn.RemoteAddr().String()
}

func (s *StratumV1Miner) GetWorkerName() string {
	return s.workerName
}

func (s *StratumV1Miner) GetConnectedAt() time.Time {
	return s.connectedAt
}

var _ StratumV1SourceConn = new(StratumV1Miner)
