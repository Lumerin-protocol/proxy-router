package connections

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"time"

	"go.uber.org/zap"
)

type Proxy struct {
	poolAddr  string
	minerAddr string
	poolUser  string
	poolPass  string

	log *zap.SugaredLogger
}

const (
	destTimeout = 5 * time.Second
)

var (
	ErrProxyWrite = errors.New("proxy-write-err")
	ErrProxyRead  = errors.New("proxy-read-err")
)

func NewProxy(clientAddr string, serverAddr string, log *zap.SugaredLogger, poolUser string, poolPass string) *Proxy {
	return &Proxy{
		poolAddr:  clientAddr,
		poolUser:  poolUser,
		poolPass:  poolPass,
		minerAddr: serverAddr,
		log:       log,
	}
}

func (p *Proxy) Run(ctx context.Context) {
	add, err := netip.ParseAddrPort(p.minerAddr)
	if err != nil {
		p.log.Panicf("Invalid address %s %w", p.poolAddr, err)
	}

	minerListener, err := net.ListenTCP("tcp", net.TCPAddrFromAddrPort(add))
	if err != nil {
		p.log.Panicf("Listener error  %s %w", p.poolAddr, err)
	}

	p.log.Infof("Stratum proxy is listening: %s", p.minerAddr)

	p.startAccepting(minerListener)
}

func (p *Proxy) startAccepting(minerListener *net.TCPListener) {
	for {
		minerConn, err := minerListener.Accept()
		if err != nil {
			p.log.Error("incoming connection accept error: %w", err)
		}

		go func(poolAddr string, minerConn net.Conn) {
			// contains all handlers on stratum level
			handlers := NewStratumHandler()

			// intercepts messages on Stratum level
			protocol := NewStratumV1(p.log, handlers)

			// Overwrites message IDs,
			// injects authorization
			// enables miner change and handshake
			manager := NewStratumV1Manager(handlers, protocol, p.log, p.poolUser, p.poolPass)
			manager.Init()

			// intercepts messages on tcp level
			proxyConn := NewProxyConn(poolAddr, minerConn, p.log, protocol)
			err := proxyConn.DialDest()

			// run only if connected
			if err == nil {
				err = proxyConn.Run(context.Background())
				if err != nil {
					p.log.Warn("Proxy error", err)
				}
			}

			err = minerConn.Close()
			if err != nil {
				p.log.Warn("Cannot close connection", err)
			}

		}(p.poolAddr, minerConn)
	}
}

func (p *Proxy) upsertConn(conn *ProxyConn) {
	// p.connections.Store(conn.ID(), conn)
}

func (p *Proxy) rmConn(conn *ProxyConn) {
	// p.connections.Delete(conn.ID())
}

func (p *Proxy) getConn(ID string) (conn *ProxyConn, ok bool) {
	// ID := GetConnID(minerAddr, poolAddr)

	// res, ok := p.connections.Load(ID)
	// if !ok {
	// 	return nil, false
	// }

	// conn, ok = res.(*ProxyConn)
	// if !ok {
	// 	p.log.DPanic("invalid data in proxy syncMap")
	// 	return nil, false
	// }

	return nil, false
}
