package connections

import (
	"context"

	"gitlab.com/TitanInd/hashrouter/mining"
	"go.uber.org/zap"
)

type StratumV1 struct {
	handler *StratumHandler
	log     *zap.SugaredLogger
}

func NewStratumV1(log *zap.SugaredLogger, handler *StratumHandler) *StratumV1 {
	return &StratumV1{
		log:     log,
		handler: handler,
	}
}

const blue = "\u001b[34m"
const green = "\u001b[32m"
const red = "\u001b[31m"
const reset = "\u001b[0m"

func (s *StratumV1) ProcessMiningMessage(ctx context.Context, msg []byte, pc *ProxyConn) []byte {
	s.log.Debugf("%sMINER    %s %s", blue, reset, msg)

	msg = s.handleMinerMsg(ctx, msg, pc)

	s.log.Debugf("%sMINER %sMOD%s %s", blue, red, reset, msg)
	return msg
}

func (s *StratumV1) ProcessPoolMessage(ctx context.Context, msg []byte, pc *ProxyConn) []byte {
	s.log.Debugf("%sPOOL     %s %s", green, reset, msg)

	msg = s.handlePoolMsg(ctx, msg, pc)

	s.log.Debugf("%sPOOL  %sMOD%s %s", green, red, reset, msg)
	return msg
}

func (s *StratumV1) handleMinerMsg(ctx context.Context, msg []byte, pc *ProxyConn) []byte {
	m, err := mining.ParseMinerMessage(msg)
	if err != nil {
		s.log.Errorf("%w", err)
		return msg
	}

	res := s.tryRunHandler(HandlerNameMinerRequest, m, pc)
	if res != nil {
		return res
	}

	switch typedMessage := m.(type) {
	case *mining.MiningSubscribe2:
		return s.tryRunHandler(HandlerNameMinerSubscribe, typedMessage, pc)
	case *mining.MiningAuthorize2:
		return s.tryRunHandler(HandlerNameMinerAuthorize, typedMessage, pc)
	case *mining.MiningSubmit:
		return s.tryRunHandler(HandlerNameMinerSubmit, typedMessage, pc)
	}

	return msg
}

func (s *StratumV1) handlePoolMsg(ctx context.Context, msg []byte, pc *ProxyConn) []byte {
	m, err := mining.ParsePoolMessage(msg)
	if err != nil {
		s.log.Errorf("Unknown miner message", string(msg))
		return msg
	}

	switch typedMessage := m.(type) {
	case *mining.MiningNotify:
		return s.tryRunHandler(HandlerNamePoolNotify, typedMessage, pc)
	case *mining.MiningSetDifficulty:
		return s.tryRunHandler(HandlerNamePoolSetDifficulty, typedMessage, pc)
	case *mining.MiningResult:
		return s.tryRunHandler(HandlerNamePoolResult, typedMessage, pc)
	}

	return msg
}

func (s *StratumV1) tryRunHandler(name HandlerName, msg mining.Message, pc *ProxyConn) []byte {
	handler, ok := s.handler.GetHandler(name)
	if !ok {
		// pass through
		return msg.Serialize()
	}
	msg = handler(msg, nil)
	return msg.Serialize()
}

func (s *StratumV1) OnClose() {}

func (s *StratumV1) ChangePool(addr string, pc *ProxyConn) error {
	return pc.ChangePool(addr)
}

type Stratum interface {
}
