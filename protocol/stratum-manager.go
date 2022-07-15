package protocol

import (
	"context"
	"errors"
	"fmt"

	"gitlab.com/TitanInd/hashrouter/protocol/message"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type StratumResultHandler = func(a message.MiningResult, s Stratum) message.MiningMessageGeneric

type StratumV1Manager struct {
	handler        *StratumHandler
	stratum        *StratumV1
	lastRequestId  *atomic.Int32
	resHandlers    map[int32]StratumResultHandler // maps MINER request id to callback, the resulting proxyrouter id can be different
	newIDToOldID   map[int]int
	isChangingPool bool
	subscribeMsg   message.MiningMessageToPool
	authUser       string
	authPass       string
	log            *zap.SugaredLogger
}

func NewStratumV1Manager(handler *StratumHandler, stratum *StratumV1, log *zap.SugaredLogger, authUser string, authPass string) *StratumV1Manager {
	return &StratumV1Manager{
		stratum:       stratum,
		handler:       handler,
		lastRequestId: atomic.NewInt32(0),
		resHandlers:   make(map[int32]StratumResultHandler),
		newIDToOldID:  make(map[int]int),
		log:           log,
		authUser:      authUser,
		authPass:      authPass,
	}
}

func (m *StratumV1Manager) Init() {
	m.handler.OnMinerRequest(func(msg message.MiningMessageToPool, s Stratum) message.MiningMessageGeneric {
		_, ok := msg.(*message.MiningSubscribe)
		if ok {
			m.subscribeMsg = msg
		}

		authMsg, ok := msg.(*message.MiningAuthorize)
		if ok {
			authMsg.SetMinerID(m.authUser)
			authMsg.SetPassword(m.authPass)
		}

		oldId := msg.GetID()
		newID := int(m.lastRequestId.Inc())
		m.newIDToOldID[newID] = oldId
		msg.SetID(newID)
		return msg
	})

	m.handler.OnPoolResult(func(msg *message.MiningResult, s Stratum) message.MiningMessageGeneric {
		newID := msg.GetID()
		handler, ok := m.resHandlers[int32(newID)]
		if ok {
			m := handler(*msg, nil)
			if m == nil {
				return nil
			}
			msg = m.(*message.MiningResult)
		}

		oldId, ok := m.newIDToOldID[newID]
		if !ok {
			m.log.Warnf("Unknown message id %d", newID)
			m.log.Warn(m.newIDToOldID)
			return msg
		}
		delete(m.newIDToOldID, newID)
		msg.SetID(oldId)
		return msg
	})
}

func (m *StratumV1Manager) SetAuth(userName string, password string) {
	m.authUser = userName
	m.authPass = password
}

func (m *StratumV1Manager) ChangePool(addr string, proxyConn Connection) error {
	m.isChangingPool = true
	defer func() { m.isChangingPool = false }()
	err := m.stratum.ChangePool(addr, proxyConn)
	if err != nil {
		return fmt.Errorf("cannot change pool %w", err)
	}
	var id int32 = 1
	m.lastRequestId.Store(id)

	_, err = m.SendPoolRequestWait(m.subscribeMsg, proxyConn)
	if err != nil {
		// TODO: on error fallback to previous pool
		return err
	}

	authMsg := message.NewMiningAuthorize()
	authMsg.SetMinerID(m.authUser)
	authMsg.SetPassword(m.authPass)

	_, err = m.SendPoolRequestWait(authMsg, proxyConn)
	if err != nil {
		// TODO: on error fallback to previous pool
		return err
	}

	return nil
}

func (m *StratumV1Manager) RegisterResultHandler(id int32, handler StratumResultHandler) {
	m.resHandlers[id] = handler
}

func (m *StratumV1Manager) SendPoolRequestWait(msg message.MiningMessageToPool, proxyConn Connection) (*message.MiningResult, error) {
	id := m.lastRequestId.Inc()
	msg.SetID(int(id))

	err := proxyConn.WriteToPool(context.TODO(), msg.Serialize())
	if err != nil {
		return nil, err
	}
	errCh := make(chan error)
	resCh := make(chan message.MiningResult)

	m.RegisterResultHandler(id, func(a message.MiningResult, s Stratum) message.MiningMessageGeneric {
		if a.IsError() {
			errCh <- errors.New(a.GetError())
		}
		resCh <- a
		return nil // do not proxy this request
	})

	select {
	case err := <-errCh:
		return nil, err
	case res := <-resCh:
		return &res, nil
	}
}
