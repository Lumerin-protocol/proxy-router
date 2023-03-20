package tcpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

// const kb = 1024 //temp

type TCPServer struct {
	serverAddr           string
	handler              ConnectionHandler
	connectionBufferSize int
	log                  interfaces.ILogger
}

func NewTCPServer(serverAddr string, connectionBufferSize int, log interfaces.ILogger) *TCPServer {
	return &TCPServer{
		serverAddr:           serverAddr,
		log:                  log,
		connectionBufferSize: connectionBufferSize,
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

	listener, err := net.Listen("tcp", add.String())

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

func (p *TCPServer) startAccepting(ctx context.Context, listener net.Listener) error {

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := listener.Accept()

		if errors.Is(err, net.ErrClosed) {
			return errors.New("incoming connection listener was closed")
		}
		if err != nil {
			p.log.Errorf("incoming connection accept error: %s", err)

			continue
		}

		if p.handler != nil {
			go func(conn net.Conn) {
				err = conn.(*net.TCPConn).SetLinger(0)

				if err != nil {
					p.log.Warnf("error updating connection linger setting: %s", err)
					return
				}

				// removed logging for each of the incoming connections (healthchecks etc)
				// HandleConnection will log errors for connections which are established from miner
				_ = p.handler.HandleConnection(ctx, conn)

				err = conn.Close()
				if err != nil {
					p.log.Warnf("error during closing connection: %s", err)
					return
				}
			}(conn)
		} else {
			conn.Close()
		}
	}
}
