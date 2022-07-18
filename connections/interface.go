package connections

import (
	"context"
	"net"
)

type ConnectionHandler interface {
	ConnectionHandler(ctx context.Context, conn net.Conn) error
}

type MessageHandler = interface {
	MinerMessageHandler(ctx context.Context, msg []byte) []byte
	PoolMessageHandler(ctx context.Context, msg []byte) []byte
}
