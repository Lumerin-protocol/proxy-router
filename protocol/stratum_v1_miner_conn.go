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
	// dependencies
	conn        net.Conn
	reader      *bufio.Reader
	log         interfaces.ILogger
	protocolLog interfaces.ILogger

	// configuration
	connectedAt   time.Time
	connTimeout   time.Duration
	workerName    string
	extraNonceMsg *stratumv1_message.MiningSubscribeResult

	// internal state
	mu        *sync.Mutex // guards isWriting
	isWriting bool        // used to temporarily pause writing messages to miner
	cond      *sync.Cond
}

func NewStratumV1MinerConn(conn net.Conn, extraNonce *stratumv1_message.MiningSubscribeResult, connectedAt time.Time, connTimeout time.Duration, log, protocolLog interfaces.ILogger) *StratumV1Miner {
	mu := new(sync.Mutex)
	return &StratumV1Miner{
		conn:          conn,
		reader:        bufio.NewReader(conn),
		connectedAt:   connectedAt,
		isWriting:     false,
		mu:            mu,
		cond:          sync.NewCond(mu),
		extraNonceMsg: extraNonce,
		connTimeout:   connTimeout,
		log:           log,
		protocolLog:   protocolLog,
	}
}

func (m *StratumV1Miner) Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error {
	return m.write(ctx, msg)
}

// write writes to miner omitting locks
func (m *StratumV1Miner) write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error {
	if m.conn == nil {
		return fmt.Errorf("miner connection is not established")
	}
	if msg == nil {
		return fmt.Errorf("nil message write attempt")
	}

	b := append(msg.Serialize(), lib.CharNewLine)

	_, err := m.conn.Write(b)
	if err != nil {
		return err
	}

	if m.protocolLog != nil {
		m.protocolLog.Debugf("=> %s", string(msg.Serialize()))
	}
	return nil
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

		if s.protocolLog != nil {
			s.protocolLog.Debugf("<= %s", string(line))
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
