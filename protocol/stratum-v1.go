package protocol

import (
	"context"

	"gitlab.com/TitanInd/hashrouter/protocol/message"
	"go.uber.org/zap"
)

// Serializes-deserializes stratum messages, invokes registered handlers
type StratumV1 struct {
	handler *StratumHandler
	log     *zap.SugaredLogger
	conn    Connection
}

func NewStratumV1(log *zap.SugaredLogger, handler *StratumHandler, conn Connection) *StratumV1 {
	stratum := &StratumV1{
		log:     log,
		handler: handler,
		conn:    conn,
	}
	conn.SetHandler(stratum)
	return stratum
}

func (s *StratumV1) MinerMessageHandler(ctx context.Context, msg []byte) []byte {
	s.log.Debugf("MINER     %s", msg)
	msg = s.handleMinerMsg(ctx, msg)
	s.log.Debugf("MINER MOD %s", msg)
	return msg
}

func (s *StratumV1) PoolMessageHandler(ctx context.Context, msg []byte) []byte {
	s.log.Debugf("POOL      %s", msg)
	msg = s.handlePoolMsg(ctx, msg)
	s.log.Debugf("POOL  MOD %s", msg)
	return msg
}

func (s *StratumV1) handleMinerMsg(ctx context.Context, msg []byte) []byte {
	m, err := message.ParseMessageToPool(msg)
	if err != nil {
		s.log.Error(err)
		return msg
	}

	res := s.tryRunHandler(HandlerNameMinerRequest, m)
	if res != nil {
		return res
	}

	switch typedMessage := m.(type) {
	case *message.MiningSubscribe:
		return s.tryRunHandler(HandlerNameMinerSubscribe, typedMessage)
	case *message.MiningAuthorize:
		return s.tryRunHandler(HandlerNameMinerAuthorize, typedMessage)
	case *message.MiningSubmit:
		return s.tryRunHandler(HandlerNameMinerSubmit, typedMessage)
	}

	return msg
}

func (s *StratumV1) handlePoolMsg(ctx context.Context, msg []byte) []byte {
	m, err := message.ParseMessageFromPool(msg)
	if err != nil {
		s.log.Errorf("Unknown miner message", string(msg))
		return msg
	}

	switch typedMessage := m.(type) {
	case *message.MiningNotify:
		return s.tryRunHandler(HandlerNamePoolNotify, typedMessage)
	case *message.MiningSetDifficulty:
		return s.tryRunHandler(HandlerNamePoolSetDifficulty, typedMessage)
	case *message.MiningResult:
		return s.tryRunHandler(HandlerNamePoolResult, typedMessage)
	}

	return msg
}

func (s *StratumV1) tryRunHandler(name HandlerName, msg message.MiningMessageGeneric) []byte {
	handler, ok := s.handler.GetHandler(name)
	if !ok {
		// pass through
		return msg.Serialize()
	}
	msg = handler(msg, s)
	if msg == nil {
		return nil
	}
	return msg.Serialize()
}

func (s *StratumV1) ChangePool(addr string) error {
	return s.conn.ChangePool(addr)
}

func (s *StratumV1) WriteToMiner(ctx context.Context, msg []byte) error {
	s.log.Debugf("WRITE TO MINER %s", string(msg))
	return s.conn.WriteToMiner(ctx, msg)
}

func (s *StratumV1) WriteToPool(ctx context.Context, msg []byte) error {
	s.log.Debugf("WRITE TO POOL %s", string(msg))
	return s.conn.WriteToPool(ctx, msg)
}
