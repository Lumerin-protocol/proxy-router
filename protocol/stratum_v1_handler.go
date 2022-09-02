package protocol

import (
	"context"

	m "gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

type HandlerName string

const (
	HandlerNameMinerRequest HandlerName = "miner-request" // Any request that comes from miner

	HandlerNameMinerSubscribe HandlerName = "miner-subscribe"
	HandlerNameMinerAuthorize HandlerName = "miner-authorize"
	HandlerNameMinerSubmit    HandlerName = "miner-submit"

	HandlerNamePoolResult        HandlerName = "pool-result" // Response to a miner request, both successful and erroneous
	HandlerNamePoolNotify        HandlerName = "pool-notify"
	HandlerNamePoolSetDifficulty HandlerName = "pool-set-difficulty"
)

// StratumHandlerCollection provides a type-safe way to register Stratum message handlers
type StratumHandlerCollection struct {
	handlers map[HandlerName]StratumSingleHandler
}

// StratumHandlerObject is passed into handler function to allow
// hook into the messaging and either modify message and propagate it to
// destination or block propagation and return response
type StratumHandlerObject interface {
	ChangePool(addr string) error
	WriteToMiner(ctx context.Context, msg []byte) error
	WriteToPool(ctx context.Context, msg []byte) error
}

func NewStratumHandler() *StratumHandlerCollection {
	return &StratumHandlerCollection{
		handlers: make(map[HandlerName]StratumSingleHandler),
	}
}

type StratumSingleHandler = func(a m.MiningMessageGeneric, s StratumHandlerObject) m.MiningMessageGeneric

func (s *StratumHandlerCollection) OnMinerRequest(handler func(msg m.MiningMessageToPool, s StratumHandlerObject) m.MiningMessageGeneric) {
	s.setHandler(HandlerNameMinerRequest, cast(handler))
}
func (s *StratumHandlerCollection) OnMinerSubscribe(handler func(msg *m.MiningSubscribe, s StratumHandlerObject) m.MiningMessageGeneric) {
	s.setHandler(HandlerNameMinerSubscribe, cast(handler))
}
func (s *StratumHandlerCollection) OnMinerAuthorize(handler func(msg *m.MiningAuthorize, s StratumHandlerObject) m.MiningMessageGeneric) {
	s.setHandler(HandlerNameMinerAuthorize, cast(handler))
}
func (s *StratumHandlerCollection) OnMinerSubmit(handler func(msg *m.MiningSubmit, s StratumHandlerObject) m.MiningMessageGeneric) {
	s.setHandler(HandlerNameMinerSubmit, cast(handler))
}
func (s *StratumHandlerCollection) OnPoolNotify(handler func(msg *m.MiningNotify, s StratumHandlerObject) m.MiningMessageGeneric) {
	s.setHandler(HandlerNamePoolNotify, cast(handler))
}
func (s *StratumHandlerCollection) OnPoolSetDifficulty(handler func(msg *m.MiningSetDifficulty, s StratumHandlerObject) m.MiningMessageGeneric) {
	s.setHandler(HandlerNamePoolSetDifficulty, cast(handler))
}
func (s *StratumHandlerCollection) OnPoolResult(handler func(msg *m.MiningResult, s StratumHandlerObject) m.MiningMessageGeneric) {
	s.setHandler(HandlerNamePoolResult, cast(handler))
}

func (s *StratumHandlerCollection) setHandler(handlerName HandlerName, handler StratumSingleHandler) {
	s.handlers[handlerName] = handler
}

func (s *StratumHandlerCollection) GetHandler(handlerName HandlerName) (StratumSingleHandler, bool) {
	handler, ok := s.handlers[handlerName]
	return handler, ok
}

// wraps handler so it can be saved to map
func cast[T m.MiningMessageGeneric](handler func(msg T, stratum StratumHandlerObject) m.MiningMessageGeneric) StratumSingleHandler {
	return func(msg m.MiningMessageGeneric, s StratumHandlerObject) m.MiningMessageGeneric {
		return handler(msg.(T), s)
	}
}
