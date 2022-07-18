package proxyhandler

import (
	"context"
	"fmt"
	"net"
	"time"

	"gitlab.com/TitanInd/hashrouter/connections"
	"gitlab.com/TitanInd/hashrouter/protocol"
	"go.uber.org/zap"
)

type ProxyHandler struct {
	poolAddr     string
	poolUser     string
	poolPassword string
	log          *zap.SugaredLogger
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
	handlers := protocol.NewStratumHandler()
	stratumV1 := protocol.NewStratumV1(p.log, handlers, proxyConn)
	proxyConn.SetHandler(stratumV1)
	manager := protocol.NewStratumV1Manager(handlers, stratumV1, p.log, p.poolUser, p.poolPassword)
	manager.Init()

	err := proxyConn.DialDest()
	if err != nil {
		return fmt.Errorf("cannot dial pool: %w", err)
	}

	go func() {
		time.Sleep(time.Second * 15)
		p.log.Info("Changing pool")

		err = manager.ChangePool("btc.f2pool.com:3333", "shev8.001", "21235365876986800")
		if err != nil {
			p.log.Errorf("Pool change error %s", err)
			return
		}
		p.log.Info("Pool changed")
	}()

	// run only if connected
	return proxyConn.Run(ctx)
}
