package tcpserver

import (
	"context"
	"net"
)

type ConnectionHandler interface {
	HandleConnection(ctx context.Context, conn net.Conn) error
}

type BufferedConn interface {
	net.Conn
	Peek(n int) ([]byte, error)
}
