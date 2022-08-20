package miner

import (
	"context"
	"net"

	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

type MinerController struct {
	defaultDest interfaces.IDestination

	repo *MinerRepo

	log interfaces.ILogger
}

func NewMinerController(defaultDest interfaces.IDestination, repo *MinerRepo, log interfaces.ILogger) *MinerController {
	return &MinerController{
		defaultDest: defaultDest,
		log:         log,
		repo:        repo,
	}
}

func (p *MinerController) HandleConnection(ctx context.Context, incomingConn net.Conn) error {
	poolPool := protocol.NewStratumV1PoolPool(p.log)
	err := poolPool.SetDest(p.defaultDest)
	if err != nil {
		p.log.Error(err)
		return err
	}
	extranonce, size := poolPool.GetExtranonce()
	msg := stratumv1_message.NewMiningSubscribeResult(extranonce, size)
	miner := protocol.NewStratumV1Miner(incomingConn, p.log, msg)
	validator := hashrate.NewHashrate(p.log, hashrate.EMA_INTERVAL)
	minerModel := protocol.NewStratumV1MinerModel(poolPool, miner, validator, p.log)

	destSplit := NewDestSplit()
	// destSplit.Allocate(30, "stratum.slushpool.com:3333", "shev8.local", "anything123")
	// destSplit.AllocateRemaining("btc.f2pool.com:3333", "shev8.001", "21235365876986800")

	minerScheduler := NewOnDemandMinerScheduler(minerModel, destSplit, p.log, nil)
	// try to connect to dest before running

	p.repo.Store(minerScheduler)

	return minerScheduler.Run(ctx)

	// return nil
}

func (p *MinerController) ChangeDestAll(dest interfaces.IDestination) error {
	p.repo.Range(func(miner MinerScheduler) bool {
		p.log.Infof("changing pool to %s for minerID %s", dest.GetHost(), miner.GetID())

		miner.Allocate(100, dest)

		return true
	})

	return nil
}
