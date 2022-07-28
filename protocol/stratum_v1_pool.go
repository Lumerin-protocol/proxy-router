package protocol

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/protocol/message"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type StratumV1Pool struct {
	conn    net.Conn
	handler *StratumHandler
	log     *zap.SugaredLogger

	msgCh         chan message.MiningMessageGeneric
	notifyMsgs    []message.MiningNotify
	setDiffMsg    *message.MiningSetDifficulty
	extraNonceMsg *message.MiningSetExtranonce

	isReading     bool
	authUser      string
	authPass      string
	lastRequestId *atomic.Uint32

	resHandlers map[int]StratumResultHandler // maps MINER request id to callback, the resulting proxyrouter id can be different
}

func NewStratumV1Pool(conn net.Conn, log *zap.SugaredLogger, authUser string, authPass string) *StratumV1Pool {
	return &StratumV1Pool{
		conn:          conn,
		msgCh:         make(chan message.MiningMessageGeneric, 100),
		notifyMsgs:    make([]message.MiningNotify, 100),
		isReading:     false, // whether messages are available to read from outside
		lastRequestId: atomic.NewUint32(0),
		resHandlers:   make(map[int]StratumResultHandler),
		log:           log,
		authUser:      authUser,
		authPass:      authPass,
	}
}

func (s *StratumV1Pool) RemoteAddr() string {
	return s.conn.RemoteAddr().String()
}

func (s *StratumV1Pool) Run(ctx context.Context) error {
	go func() {
		err := s.run(ctx)
		s.log.Error(err)
	}()
	return nil
}

func (s *StratumV1Pool) run(ctx context.Context) error {
	sourceReader := bufio.NewReader(s.conn)
	s.log.Debug("pool reader started")

	for {
		// line, isPrefix, err := sourceReader.ReadLine()
		// if isPrefix {
		// 	panic("line is too long")
		// }
		line, err := sourceReader.ReadBytes('\n')

		if err != nil {
			return err
		}

		lib.LogMsg(false, true, line, s.log)

		m, err := message.ParseMessageFromPool(line)
		if err != nil {
			s.log.Errorf("Unknown miner message", string(line))
		}

		switch typedMessage := m.(type) {
		case *message.MiningNotify:
			if typedMessage.GetCleanJobs() {
				s.notifyMsgs = s.notifyMsgs[:0]
			}
			s.notifyMsgs = append(s.notifyMsgs, *typedMessage)

		case *message.MiningSetDifficulty:
			s.setDiffMsg = typedMessage

		case *message.MiningResult:
			id := typedMessage.GetID()
			handler, ok := s.resHandlers[id]
			if ok {
				handledMsg := handler(*typedMessage)
				if handledMsg != nil {
					m = handledMsg.(*message.MiningResult)
				}
			}
		}

		if s.isReading {
			s.msgCh <- m
		} else {
			s.log.Debugf("not reading")
		}
	}
}

// Allows to connect to a new pool
func (m *StratumV1Pool) Connect() error {
	err := m.Run(context.Background())
	if err != nil {
		return err
	}
	subscribeRes, err := m.SendPoolRequestWait(message.NewMiningSubscribe(1, "miner", "1"))
	if err != nil {
		// TODO: on error fallback to previous pool
		return err
	}
	if subscribeRes.IsError() {
		return fmt.Errorf("invalid subscribe response %s", subscribeRes.Serialize())
	}
	m.log.Debug("connect: subscribe sent")

	data := [3]interface{}{}

	err = json.Unmarshal(subscribeRes.Result, &data)
	if err != nil {
		return fmt.Errorf("cannot unmarhal subscribe response %s %w", subscribeRes.Serialize(), err)
	}
	extranonce, extranonceSize := data[1].(string), int(data[2].(float64))
	msg := message.NewMiningSetExtranonce()
	msg.SetExtranonce(extranonce, extranonceSize)
	m.extraNonceMsg = msg

	authMsg := message.NewMiningAuthorize(1, m.authUser, m.authPass)
	_, err = m.SendPoolRequestWait(authMsg)
	if err != nil {
		m.log.Debugf("reconnect: error sent subscribe %w", err)

		// TODO: on error fallback to previous pool
		return err
	}
	m.log.Debug("connect: authorize sent")

	m.resendRelevantNotifications(context.TODO())
	m.isReading = true

	return nil
}

func (m *StratumV1Pool) SendPoolRequestWait(msg message.MiningMessageToPool) (*message.MiningResult, error) {
	id := int(m.lastRequestId.Inc())
	msg.SetID(int(id))

	err := m.Write(context.TODO(), msg)
	if err != nil {
		return nil, err
	}
	errCh := make(chan error)
	resCh := make(chan message.MiningResult)

	m.RegisterResultHandler(id, func(a message.MiningResult) message.MiningMessageGeneric {
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
	}
}

func (m *StratumV1Pool) RegisterResultHandler(id int, handler StratumResultHandler) {
	m.resHandlers[id] = handler
}

func (m *StratumV1Pool) ResendRelevantNotifications(ctx context.Context) {
	m.isReading = false
	m.resendRelevantNotifications(ctx)
	m.isReading = true
}

func (m *StratumV1Pool) resendRelevantNotifications(ctx context.Context) {
	m.msgCh <- m.extraNonceMsg
	m.log.Infof("extranonce was resent")

	m.msgCh <- m.setDiffMsg
	m.log.Infof("set-difficulty was resent")
	for _, msg := range m.notifyMsgs {
		m.msgCh <- &msg
	}
	m.log.Infof("notify messages (%d) were resent", len(m.notifyMsgs))
}

func (s *StratumV1Pool) GetChan() <-chan message.MiningMessageGeneric {
	return s.msgCh
}

func (s *StratumV1Pool) Read() (message.MiningMessageGeneric, error) {
	msg := <-s.GetChan()
	return msg, nil
}

func (m *StratumV1Pool) Write(ctx context.Context, msg message.MiningMessageGeneric) error {
	switch typedMsg := msg.(type) {
	case *message.MiningSubmit:
		typedMsg.SetWorkerName(m.authUser)
		msg = typedMsg
	}
	lib.LogMsg(false, false, msg.Serialize(), m.log)
	b := fmt.Sprintf("%s\n", msg.Serialize())
	_, err := m.conn.Write([]byte(b))
	return err
}

func (m *StratumV1Pool) GetExtranonce() (string, int) {
	return m.extraNonceMsg.GetExtranonce()
}
