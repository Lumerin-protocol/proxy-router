package protocol

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
	"go.uber.org/atomic"
)

// StratumV1PoolConn represents connection to the pool on the protocol level
type StratumV1PoolConn struct {
	authUser string // pool auth
	authPass string // pool auth

	conn net.Conn // tcp connection

	notifyMsgs    []stratumv1_message.MiningNotify       // recent relevant notify messages, (respects stratum clean_jobs flag)
	setDiffMsg    *stratumv1_message.MiningSetDifficulty // recent difficulty message
	extraNonceMsg *stratumv1_message.MiningSetExtranonce // keeps relevant extranonce (picked from mining.subscribe response)
	// TODO: handle pool setExtranonce message

	msgCh     chan stratumv1_message.MiningMessageGeneric // auxillary channel to relay messages
	isReading bool                                        // if false messages will not be availabe to read from outside, used for authentication handshake

	lastRequestId *atomic.Uint32                 // stratum request id counter
	resHandlers   map[int]StratumV1ResultHandler // allows to register callbacks for particular messages to simplify transaction flow

	log interfaces.ILogger
}

func NewStratumV1Pool(conn net.Conn, log interfaces.ILogger, authUser string, authPass string) *StratumV1PoolConn {
	return &StratumV1PoolConn{
		conn:          conn,
		msgCh:         make(chan stratumv1_message.MiningMessageGeneric, 100),
		notifyMsgs:    make([]stratumv1_message.MiningNotify, 100),
		isReading:     false,
		lastRequestId: atomic.NewUint32(0),
		resHandlers:   make(map[int]StratumV1ResultHandler),
		log:           log,
		authUser:      authUser,
		authPass:      authPass,
	}
}

func (s *StratumV1PoolConn) RemoteAddr() string {
	return s.conn.RemoteAddr().String()
}

func (s *StratumV1PoolConn) Run(ctx context.Context) error {
	go func() {
		err := s.run(ctx)
		s.log.Error(err)
	}()
	return nil
}

func (s *StratumV1PoolConn) run(ctx context.Context) error {
	sourceReader := bufio.NewReader(s.conn)
	s.log.Debug("pool reader started")

	for {
		line, isPrefix, err := sourceReader.ReadLine()
		if isPrefix {
			panic("line is too long")
		}

		if err != nil {
			return err
		}

		lib.LogMsg(false, true, line, s.log)

		m, err := stratumv1_message.ParseMessageFromPool(line)
		if err != nil {
			s.log.Errorf("Unknown miner message", string(line))
		}

		switch typedMessage := m.(type) {
		case *stratumv1_message.MiningNotify:
			if typedMessage.GetCleanJobs() {
				s.notifyMsgs = s.notifyMsgs[:0]
			}
			s.notifyMsgs = append(s.notifyMsgs, *typedMessage)

		case *stratumv1_message.MiningSetDifficulty:
			s.setDiffMsg = typedMessage

		case *stratumv1_message.MiningResult:
			id := typedMessage.GetID()
			handler, ok := s.resHandlers[id]
			if ok {
				handledMsg := handler(*typedMessage)
				if handledMsg != nil {
					m = handledMsg.(*stratumv1_message.MiningResult)
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
func (m *StratumV1PoolConn) Connect() error {
	err := m.Run(context.Background())
	if err != nil {
		return err
	}
	subscribeRes, err := m.sendPoolRequestWait(stratumv1_message.NewMiningSubscribe(1, "miner", "1"))
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
	msg := stratumv1_message.NewMiningSetExtranonce()
	msg.SetExtranonce(extranonce, extranonceSize)
	m.extraNonceMsg = msg

	authMsg := stratumv1_message.NewMiningAuthorize(1, m.authUser, m.authPass)
	_, err = m.sendPoolRequestWait(authMsg)
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

// sendPoolRequestWait sends a message and awaits for the response
func (m *StratumV1PoolConn) sendPoolRequestWait(msg stratumv1_message.MiningMessageToPool) (*stratumv1_message.MiningResult, error) {
	id := int(m.lastRequestId.Inc())
	msg.SetID(int(id))

	err := m.Write(context.TODO(), msg)
	if err != nil {
		return nil, err
	}
	errCh := make(chan error)
	resCh := make(chan stratumv1_message.MiningResult)

	m.registerResultHandler(id, func(a stratumv1_message.MiningResult) stratumv1_message.MiningMessageGeneric {
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

func (m *StratumV1PoolConn) registerResultHandler(id int, handler StratumV1ResultHandler) {
	m.resHandlers[id] = handler
}

// Pauses emitting any pool messages, then sends cached messages for a recent job, and then resumes pool message flow
func (m *StratumV1PoolConn) ResendRelevantNotifications(ctx context.Context) {
	m.isReading = false
	m.resendRelevantNotifications(ctx)
	m.isReading = true
}

// resendRelevantNotifications sends cached extranonce, set_difficulty and notify messages
// useful after changing miner's destinations
func (m *StratumV1PoolConn) resendRelevantNotifications(ctx context.Context) {
	m.msgCh <- m.extraNonceMsg
	m.log.Infof("extranonce was resent")

	m.msgCh <- m.setDiffMsg
	m.log.Infof("set-difficulty was resent")
	for _, msg := range m.notifyMsgs {
		m.msgCh <- &msg
	}
	m.log.Infof("notify messages (%d) were resent", len(m.notifyMsgs))
}

// Read reads message from pool
func (s *StratumV1PoolConn) Read() (stratumv1_message.MiningMessageGeneric, error) {
	msg := <-s.msgCh
	return msg, nil
}

// Write writes message to pool
func (m *StratumV1PoolConn) Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error {
	switch typedMsg := msg.(type) {
	case *stratumv1_message.MiningSubmit:
		typedMsg.SetWorkerName(m.authUser)
		msg = typedMsg
	}
	lib.LogMsg(false, false, msg.Serialize(), m.log)
	b := fmt.Sprintf("%s\n", msg.Serialize())
	_, err := m.conn.Write([]byte(b))
	return err
}

// Returns current extranonce values
func (m *StratumV1PoolConn) GetExtranonce() (string, int) {
	return m.extraNonceMsg.GetExtranonce()
}
