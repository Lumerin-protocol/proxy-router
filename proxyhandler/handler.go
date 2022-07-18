package proxyhandler

import (
	"context"
	"fmt"
	"net"
	"sync"

	"gitlab.com/TitanInd/hashrouter/connections"
	"gitlab.com/TitanInd/hashrouter/protocol"
	"go.uber.org/zap"
)

type ProxyHandler struct {
	poolAddr     string
	poolUser     string
	poolPassword string
	log          *zap.SugaredLogger
	miners       sync.Map
}

func NewProxyHandler(poolAddr string, poolUser string, poolPassword string, log *zap.SugaredLogger) *ProxyHandler {
	return &ProxyHandler{
		poolAddr:     poolAddr,
		poolUser:     poolUser,
		poolPassword: poolPassword,
		log:          log,
	}
}

func (p *ProxyHandler) ConnectionHandler(ctx context.Context, incomingConn net.Conn) error {
	// connection-scoped objects
	proxyConn := connections.NewProxyConn(p.poolAddr, incomingConn, p.log)
	//------------------------------
	handlers := protocol.NewStratumHandler()
	stratumV1 := protocol.NewStratumV1(p.log, handlers, proxyConn)
	proxyConn.SetHandler(stratumV1)
	manager := protocol.NewStratumV1Manager(handlers, stratumV1, p.log, p.poolUser, p.poolPassword)
	manager.Init()

	err := proxyConn.DialDest()
	if err != nil {
		return fmt.Errorf("cannot dial pool: %w", err)
	}

	p.miners.Store(manager.GetID(), manager)

	// run only if connected
	return proxyConn.Run(ctx)
}

func (p *ProxyHandler) ChangeDest(addr string, username string, pwd string) error {

	p.miners.Range(func(key, value any) bool {
		manager, ok := value.(*protocol.StratumV1Manager)
		if !ok {
			panic("invalid type")
		}

		p.log.Infof("changing pool to %s for minerID %s", addr, manager.GetID())
		err := manager.ChangePool(addr, username, pwd)
		if err != nil {
			p.log.Errorf("error changing pool %w", err)
		} else {
			p.log.Info("Pool changed for minerid %s", manager.GetID())
		}
		return true
	})

	return nil

}
