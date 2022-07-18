package tcpserver

import (
	"context"
	"net"
)

type ConnectionHandler interface {
	ConnectionHandler(ctx context.Context, conn net.Conn) error
}
