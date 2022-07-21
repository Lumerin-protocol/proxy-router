package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gitlab.com/TitanInd/hashrouter/protocol/message"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type StratumResultHandler = func(a message.MiningResult) message.MiningMessageGeneric

// StratumV1Manager injects authorization, enables pool change and
// then sends handshake messages, conrols message ids
type StratumV1Manager struct {
	handler        *StratumHandler
	stratum        *StratumV1
	lastRequestId  *atomic.Uint32
	resHandlers    map[int]StratumResultHandler // maps MINER request id to callback, the resulting proxyrouter id can be different
	isChangingPool bool
	subscribeMsg   message.MiningMessageToPool
	authUser       string
	authPass       string
	log            *zap.SugaredLogger

	notificationsOnHold           chan message.MiningMessageGeneric
	notificationsCh               chan message.MiningMessageGeneric
	notificationsPassthroughState chan bool
}

func NewStratumV1Manager(handler *StratumHandler, stratum *StratumV1, log *zap.SugaredLogger, authUser string, authPass string) *StratumV1Manager {
	return &StratumV1Manager{
		stratum:                       stratum,
		handler:                       handler,
		lastRequestId:                 atomic.NewUint32(0),
		resHandlers:                   make(map[int]StratumResultHandler),
		log:                           log,
		authUser:                      authUser,
		authPass:                      authPass,
		notificationsOnHold:           make(chan message.MiningMessageGeneric, 100),
		notificationsCh:               make(chan message.MiningMessageGeneric, 100),
		notificationsPassthroughState: make(chan bool, 1),
	}
}

func (m *StratumV1Manager) Init() {
	go func() {
		m.NotificationsPassthrough(context.TODO())
	}()

	m.handler.OnMinerRequest(func(msg message.MiningMessageToPool, s StratumHandlerObject) message.MiningMessageGeneric {
		_, ok := msg.(*message.MiningSubscribe)
		if ok {
			m.subscribeMsg = msg
		}

		authMsg, ok := msg.(*message.MiningAuthorize)
		if ok {
			authMsg.SetMinerID(m.authUser)
			authMsg.SetPassword(m.authPass)
		}

		submitMsg, ok := msg.(*message.MiningSubmit)
		if ok {
			submitMsg.SetWorkerName(m.authUser)
		}

		return msg
	})

	m.handler.OnPoolResult(func(msg *message.MiningResult, s StratumHandlerObject) message.MiningMessageGeneric {
		if msg.IsError() {
			m.log.Errorf("POOL  ERR %s", msg.Serialize())
		}

		id := msg.GetID()
		handler, ok := m.resHandlers[id]
		if ok {
			m := handler(*msg)
			if m == nil {
				return nil
			}
			msg = m.(*message.MiningResult)
		}

		return msg
	})

	m.handler.OnPoolNotify(func(msg *message.MiningNotify, s StratumHandlerObject) message.MiningMessageGeneric {
		m.notificationsCh <- msg
		return nil
	})

	m.handler.OnPoolSetDifficulty(func(msg *message.MiningSetDifficulty, s StratumHandlerObject) message.MiningMessageGeneric {
		m.notificationsCh <- msg
		return nil
	})
}

func (m *StratumV1Manager) GetID() string {
	return m.stratum.conn.GetMinerIP()
}

func (m *StratumV1Manager) SetAuth(userName string, password string) {
	m.authUser = userName
	m.authPass = password
}

func (m *StratumV1Manager) ChangePool(addr string, username string, password string) error {
	m.HoldNotifications(context.TODO())
	err := m.stratum.ChangePool(addr)
	if err != nil {
		return fmt.Errorf("cannot change pool %w", err)
	}

	m.SetAuth(username, password)

	subscribeRes, err := m.SendPoolRequestWait(m.subscribeMsg)
	if err != nil {
		// TODO: on error fallback to previous pool
		return err
	}
	if subscribeRes.IsError() {
		return fmt.Errorf("invalid subscribe response %s", subscribeRes.Serialize())
	}
	m.log.Debug("change pool: subscribe sent")

	data := [3]interface{}{}
	// subscribeRes.Result.
	err = json.Unmarshal(subscribeRes.Result, &data)
	if err != nil {
		return fmt.Errorf("cannot unmarhal subscribe response %s %w", subscribeRes.Serialize(), err)
	}
	extranonce, extranonceSize := data[1].(string), int(data[2].(float64))
	msg := message.NewMiningSetExtranonce()
	msg.SetExtranonce(extranonce, extranonceSize)
	err = m.SendNotice(msg)
	if err != nil {
		return err
	}

	m.log.Infof("change pool: extranonce sent %s %d", extranonce, extranonceSize)

	authMsg := message.NewMiningAuthorize()
	authMsg.SetMinerID(username)
	authMsg.SetPassword(password)

	_, err = m.SendPoolRequestWait(authMsg)
	if err != nil {
		m.log.Debugf("reconnect: error sent subscribe %w", err)

		// TODO: on error fallback to previous pool
		return err
	}
	m.log.Debug("change pool: authorize sent")

	m.ReleaseNotifications(context.TODO())

	m.isChangingPool = false
	m.log.Debug("change pool: finished")

	return nil
}

func (m *StratumV1Manager) HoldNotifications(ctx context.Context) {
	m.notificationsPassthroughState <- false
	m.log.Info("notifications put on-hold")
}

func (m *StratumV1Manager) ReleaseNotifications(ctx context.Context) error {
	m.log.Infof("on-hold notifications to be released: %d", len(m.notificationsCh))
	m.notificationsPassthroughState <- true
	m.log.Info("on-hold notifications were released")
	return nil
}

func (m *StratumV1Manager) RegisterResultHandler(id int, handler StratumResultHandler) {
	m.resHandlers[id] = handler
}

func (m *StratumV1Manager) SendNotice(msg message.MiningMessageGeneric) error {
	return m.stratum.WriteToMiner(context.TODO(), msg.Serialize())
}

func (m *StratumV1Manager) SendPoolRequestWait(msg message.MiningMessageToPool) (*message.MiningResult, error) {
	id := int(m.lastRequestId.Inc())
	msg.SetID(int(id))

	err := m.stratum.WriteToPool(context.TODO(), msg.Serialize())
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

func (m *StratumV1Manager) NotificationsPassthrough(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-m.notificationsCh:
			err := m.stratum.WriteToMiner(context.TODO(), msg.Serialize())
			if err != nil {
				return err
			}
		case state := <-m.notificationsPassthroughState:
			if !state {
				m.log.Info("notifications passthrough paused")
				for {
					state := <-m.notificationsPassthroughState
					if state {
						m.log.Info("notifications passthrough resumed")
						break
					}
				}
			}
		}
	}
}
