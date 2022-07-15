package connections

import m "gitlab.com/TitanInd/hashrouter/mining"

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

type StratumSingleHandler = func(a m.Message, s Stratum) m.Message

func (s *StratumHandler) OnMinerRequest(handler func(msg m.MinerMessage, s Stratum) m.Message) {
	s.setHandler(HandlerNameMinerRequest, cast(handler))
}
func (s *StratumHandler) OnMinerSubscribe(handler func(msg *m.MiningSubscribe2, s Stratum) m.Message) {
	s.setHandler(HandlerNameMinerSubscribe, cast(handler))
}
func (s *StratumHandler) OnMinerAuthorize(handler func(msg *m.MiningAuthorize2, s Stratum) m.Message) {
	s.setHandler(HandlerNameMinerAuthorize, cast(handler))
}
func (s *StratumHandler) OnMinerSubmit(handler func(msg *m.MiningSubmit, s Stratum) m.Message) {
	s.setHandler(HandlerNameMinerSubmit, cast(handler))
}
func (s *StratumHandler) OnPoolNotify(handler func(msg *m.MiningNotify, s Stratum) m.Message) {
	s.setHandler(HandlerNamePoolNotify, cast(handler))
}
func (s *StratumHandler) OnPoolSetDifficulty(handler func(msg *m.MiningSetDifficulty, s Stratum) m.Message) {
	s.setHandler(HandlerNamePoolSetDifficulty, cast(handler))
}
func (s *StratumHandler) OnPoolResult(handler func(msg *m.MiningResult, s Stratum) m.Message) {
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
func cast[T m.Message](handler func(msg T, stratum Stratum) m.Message) StratumSingleHandler {
	return func(msg m.Message, s Stratum) m.Message {
		return handler(msg.(T), s)
	}
}
