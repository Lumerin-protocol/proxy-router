package connections

import (
	"errors"
	"fmt"

	"gitlab.com/TitanInd/hashrouter/mining"
	m "gitlab.com/TitanInd/hashrouter/mining"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type StratumV1Manager struct {
	handler        *StratumHandler
	stratum        *StratumV1
	lastRequestId  *atomic.Int32
	resHandlers    map[int32]StratumResultHandler // maps MINER request id to callback, the resulting proxyrouter id can be different
	newIDToOldID   map[int]int
	isChangingPool bool
	subscribeMsg   mining.MinerMessage
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
	m.handler.OnMinerRequest(func(msg mining.MinerMessage, s Stratum) mining.Message {
		_, ok := msg.(*mining.MiningSubscribe2)
		if ok {
			m.subscribeMsg = msg
		}

		authMsg, ok := msg.(*mining.MiningAuthorize2)
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

	m.handler.OnPoolResult(func(msg *mining.MiningResult, s Stratum) mining.Message {
		newID := msg.GetID()
		handler, ok := m.resHandlers[int32(newID)]
		if ok {
			m := handler(*msg, nil)
			if m == nil {
				return nil
			}
			msg = m.(*mining.MiningResult)
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

func (m *StratumV1Manager) ChangePool(addr string, proxyConn *ProxyConn) error {
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

	authMsg := mining.NewMiningAuthorizeMsg(m.authUser, m.authPass)
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

func (m *StratumV1Manager) SendPoolRequestWait(msg mining.MinerMessage, proxyConn *ProxyConn) (*mining.MiningResult, error) {
	id := m.lastRequestId.Inc()
	msg.SetID(int(id))

	_, err := proxyConn.poolConn.Write(msg.Serialize())
	if err != nil {
		return nil, err
	}
	errCh := make(chan error)
	resCh := make(chan mining.MiningResult)

	m.RegisterResultHandler(id, func(a mining.MiningResult, s Stratum) mining.Message {
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

type StratumResultHandler = func(a mining.MiningResult, s Stratum) m.Message
