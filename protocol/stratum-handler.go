package protocol

import m "gitlab.com/TitanInd/hashrouter/protocol/message"

type HandlerName string

const (
	HandlerNameMinerRequest   HandlerName = "miner-request"
	HandlerNameMinerSubscribe HandlerName = "miner-subscribe"
	HandlerNameMinerAuthorize HandlerName = "miner-authorize"
	HandlerNameMinerSubmit    HandlerName = "miner-submit"

	HandlerNamePoolResult        HandlerName = "pool-result"
	HandlerNamePoolNotify        HandlerName = "pool-notify"
	HandlerNamePoolSetDifficulty HandlerName = "pool-set-difficulty"
)

type StratumHandler struct {
	handlers map[HandlerName]StratumSingleHandler
}

func NewStratumHandler() *StratumHandler {
	return &StratumHandler{
		handlers: make(map[HandlerName]StratumSingleHandler),
	}
}

type StratumSingleHandler = func(a m.MiningMessageGeneric, s Stratum) m.MiningMessageGeneric

func (s *StratumHandler) OnMinerRequest(handler func(msg m.MiningMessageToPool, s Stratum) m.MiningMessageGeneric) {
	s.setHandler(HandlerNameMinerRequest, cast(handler))
}
func (s *StratumHandler) OnMinerSubscribe(handler func(msg *m.MiningSubscribe, s Stratum) m.MiningMessageGeneric) {
	s.setHandler(HandlerNameMinerSubscribe, cast(handler))
}
func (s *StratumHandler) OnMinerAuthorize(handler func(msg *m.MiningAuthorize, s Stratum) m.MiningMessageGeneric) {
	s.setHandler(HandlerNameMinerAuthorize, cast(handler))
}
func (s *StratumHandler) OnMinerSubmit(handler func(msg *m.MiningSubmit, s Stratum) m.MiningMessageGeneric) {
	s.setHandler(HandlerNameMinerSubmit, cast(handler))
}
func (s *StratumHandler) OnPoolNotify(handler func(msg *m.MiningNotify, s Stratum) m.MiningMessageGeneric) {
	s.setHandler(HandlerNamePoolNotify, cast(handler))
}
func (s *StratumHandler) OnPoolSetDifficulty(handler func(msg *m.MiningSetDifficulty, s Stratum) m.MiningMessageGeneric) {
	s.setHandler(HandlerNamePoolSetDifficulty, cast(handler))
}
func (s *StratumHandler) OnPoolResult(handler func(msg *m.MiningResult, s Stratum) m.MiningMessageGeneric) {
	s.setHandler(HandlerNamePoolResult, cast(handler))
}

func (s *StratumHandler) setHandler(handlerName HandlerName, handler StratumSingleHandler) {
	s.handlers[handlerName] = handler
}

func (s *StratumHandler) GetHandler(handlerName HandlerName) (StratumSingleHandler, bool) {
	handler, ok := s.handlers[handlerName]
	return handler, ok
}

// wraps handler so it can be saved to map
func cast[T m.MiningMessageGeneric](handler func(msg T, stratum Stratum) m.MiningMessageGeneric) StratumSingleHandler {
	return func(msg m.MiningMessageGeneric, s Stratum) m.MiningMessageGeneric {
		return handler(msg.(T), s)
	}
}
