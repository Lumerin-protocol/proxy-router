package tcpserver

import (
	"context"
	"net"
)

type ConnectionHandler interface {
	HandleConnection(ctx context.Context, conn net.Conn) error
}
