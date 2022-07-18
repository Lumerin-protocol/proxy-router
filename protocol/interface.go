package protocol

import "context"

// StratumHandlerObject is passed into handler function to allow
// hook into the messaging and either modify message and propagate it to
// destination or block propagation and return response
type StratumHandlerObject interface {
	ChangePool(addr string) error
	WriteToMiner(ctx context.Context, msg []byte) error
	WriteToPool(ctx context.Context, msg []byte) error
}

type Connection interface {
	SetHandler(pc MessageHandler)
	ChangePool(addr string) error
	GetMinerIP() string
	WriteToPool(ctx context.Context, bytes []byte) error
	WriteToMiner(ctx context.Context, bytes []byte) error
}

type MessageHandler = interface {
	MinerMessageHandler(ctx context.Context, msg []byte) []byte
	PoolMessageHandler(ctx context.Context, msg []byte) []byte
}
