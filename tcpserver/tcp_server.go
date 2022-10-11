package tcpserver

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type TCPServer struct {
	serverAddr string
	handler    ConnectionHandler

	log interfaces.ILogger
}

func NewTCPServer(serverAddr string, log interfaces.ILogger) *TCPServer {
	return &TCPServer{
		serverAddr: serverAddr,
		log:        log,
	}
}

func (p *TCPServer) SetConnectionHandler(handler ConnectionHandler) {
	p.handler = handler
}

func (p *TCPServer) Run(ctx context.Context) error {
	add, err := netip.ParseAddrPort(p.serverAddr)
	if err != nil {
		return fmt.Errorf("invalid server address %s %w", p.serverAddr, err)
	}

	listener, err := net.ListenTCP("tcp", net.TCPAddrFromAddrPort(add))
	if err != nil {
		return fmt.Errorf("listener error %s %w", p.serverAddr, err)
	}

	p.log.Infof("tcp server is listening: %s", p.serverAddr)

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- p.startAccepting(ctx, listener)
	}()

	select {
	case <-ctx.Done():
		err := listener.Close()
		if err != nil {
			return err
		}
		err = ctx.Err()
		p.log.Infof("tcp server closed: %s", p.serverAddr)
		return err
	case err = <-serverErr:
	}

	return err
}

func (p *TCPServer) startAccepting(ctx context.Context, listener *net.TCPListener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			p.log.Errorf("incoming connection accept error: %s", err)
			continue
		}

		if p.handler != nil {
			go func(conn net.Conn) {
				err := p.handler.HandleConnection(ctx, conn)
				if err != nil {
					p.log.Warn("connection handler error: %w", err)
				}

				err = conn.Close()
				if err != nil {
					p.log.Warn("error during closing connection", err)
					return
				}

				p.log.Info("connection closed: %s", conn.RemoteAddr())

			}(conn)
		}
	}
}
