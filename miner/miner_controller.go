package miner

import (
	"context"
	"fmt"
	"net"

	"gitlab.com/TitanInd/hashrouter/connections"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol"
)

type MinerController struct {
	poolAddr     string
	poolUser     string
	poolPassword string

	repo *MinerRepo

	log interfaces.ILogger
}

func NewMinerController(poolAddr string, poolUser string, poolPassword string, repo *MinerRepo, log interfaces.ILogger) *MinerController {
	return &MinerController{
		poolAddr:     poolAddr,
		poolUser:     poolUser,
		poolPassword: poolPassword,
		log:          log,
		repo:         repo,
	}
}

func (p *MinerController) HandleConnection(ctx context.Context, incomingConn net.Conn) error {
	// connection-scoped objects
	proxyConn := connections.NewProxyConn(p.poolAddr, incomingConn, p.log)
	//------------------------------
	handlers := protocol.NewStratumHandlerCollection()
	stratumV1 := protocol.NewStratumV1(p.log, handlers, proxyConn)
	proxyConn.SetHandler(stratumV1)
	manager := protocol.NewStratumV1Manager(handlers, stratumV1, p.log, p.poolUser, p.poolPassword)
	manager.Init()

	// try to connect to dest before running
	err := proxyConn.DialDest()
	if err != nil {
		return fmt.Errorf("cannot dial pool: %w", err)
	}

	p.repo.Store(manager)

	return proxyConn.Run(ctx)
}

func (p *MinerController) ChangeDestAll(addr string, username string, pwd string) error {
	p.repo.Range(func(miner Miner) bool {
		p.log.Infof("changing pool to %s for minerID %s", addr, miner.GetID())

		err := miner.ChangePool(addr, username, pwd)
		if err != nil {
			p.log.Errorf("error changing pool %w", err)
		} else {
			p.log.Info("Pool changed for minerid %s", miner.GetID())
		}
		return true
	})

	return nil
}
