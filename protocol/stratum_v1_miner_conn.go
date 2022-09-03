package protocol

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

type StratumV1Miner struct {
	conn   net.Conn
	reader *bufio.Reader

	isWriting bool        // used to temporarily pause writing messages to miner
	mu        *sync.Mutex // guards isWriting

	cond          *sync.Cond
	extraNonceMsg *stratumv1_message.MiningSubscribeResult
	workerName    string
	log           interfaces.ILogger
}

func NewStratumV1Miner(conn net.Conn, log interfaces.ILogger, extraNonce *stratumv1_message.MiningSubscribeResult) *StratumV1Miner {
	mu := new(sync.Mutex)
	return &StratumV1Miner{
		conn:          conn,
		reader:        bufio.NewReader(conn),
		isWriting:     false,
		mu:            mu,
		cond:          sync.NewCond(mu),
		extraNonceMsg: extraNonce,
		log:           log,
	}
}

func (m *StratumV1Miner) Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error {
	m.waitForWriteUnlock()
	return m.write(ctx, msg)
}

// write writes to miner omitting locks
func (m *StratumV1Miner) write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error {
	lib.LogMsg(true, false, m.conn.RemoteAddr().String(), msg.Serialize(), m.log)

	b := fmt.Sprintf("%s\n", msg.Serialize())
	_, err := m.conn.Write([]byte(b))
	return err
}

func (s *StratumV1Miner) Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error) {
	for {
		line, isPrefix, err := s.reader.ReadLine()
		if isPrefix {
			panic("line is too long")
		}

		if err != nil {
			return nil, err
		}

		lib.LogMsg(true, true, s.conn.RemoteAddr().String(), line, s.log)

		m, err := stratumv1_message.ParseMessageToPool(line)

		if err != nil {
			s.log.Errorf("unknown miner message: %s", string(line))
			continue
		}

		switch typedMessage := m.(type) {
		case *stratumv1_message.MiningSubscribe:
			s.extraNonceMsg.SetID(m.GetID())
			err := s.write(context.TODO(), s.extraNonceMsg)
			if err != nil {
				return nil, err
			}
			continue

		case *stratumv1_message.MiningAuthorize:
			s.setWorkerName(typedMessage.GetWorkerName())

			msg, _ := stratumv1_message.ParseMiningResult([]byte(`{"id":47,"result":true,"error":null}`))
			msg.SetID(m.GetID())
			err := s.write(context.TODO(), msg)
			if err != nil {
				return nil, err
			}
			s.writeUnlock()

			continue
		}

		return m, nil
	}
}

func (s *StratumV1Miner) GetID() string {
	return s.conn.RemoteAddr().String()
}

func (s *StratumV1Miner) setWorkerName(name string) {
	s.workerName = name
}

func (s *StratumV1Miner) GetWorkerName() string {
	return s.workerName
}

func (s *StratumV1Miner) writeUnlock() {
	s.mu.Lock()
	s.isWriting = true
	s.mu.Unlock()
	s.log.Debug("writing to miner is released")
	s.cond.Broadcast()
}

func (m *StratumV1Miner) waitForWriteUnlock() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for !m.isWriting {
		m.log.Info("writing to miner is locked")
		m.cond.Wait()
	}
}

var _ StratumV1SourceConn = new(StratumV1Miner)
