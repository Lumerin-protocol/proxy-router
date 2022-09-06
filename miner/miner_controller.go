package miner

import (
	"context"
	"net"
	"strings"

	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
	"gitlab.com/TitanInd/hashrouter/tcpserver"
)

type MinerController struct {
	defaultDest interfaces.IDestination
	collection  interfaces.ICollection[MinerScheduler]
	log         interfaces.ILogger
}

func NewMinerController(defaultDest interfaces.IDestination, collection interfaces.ICollection[MinerScheduler], log interfaces.ILogger) *MinerController {
	return &MinerController{
		defaultDest: defaultDest,
		log:         log,
		collection:  collection,
	}
}

func (p *MinerController) HandleConnection(ctx context.Context, incomingConn net.Conn) error {
	p.log.Infof("incoming miner connection: %s", incomingConn.RemoteAddr().String())
	// TODO: peek if incoming connection is stratum connection

	buffered := tcpserver.NewBufferedConn(incomingConn)
	bytes, err := buffered.Peek(50)
	p.log.Error(err)
	peakedMsg := strings.ToLower(string(bytes))

	if !(strings.Contains(peakedMsg, "id") &&
		strings.Contains(peakedMsg, "mining") &&
		strings.Contains(peakedMsg, "params")) {
		p.log.Infof("invalid incoming message: %s", peakedMsg)
		err := buffered.Close()
		return err
	}

	incomingConn = buffered

	poolPool := protocol.NewStratumV1PoolPool(p.log.Named(incomingConn.RemoteAddr().String()))
	err = poolPool.SetDest(p.defaultDest, nil)
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

	minerScheduler := NewOnDemandMinerScheduler(minerModel, destSplit, p.log, p.defaultDest)
	// try to connect to dest before running

	p.collection.Store(minerScheduler)

	return minerScheduler.Run(ctx)

	// return nil
}

func (p *MinerController) ChangeDestAll(dest interfaces.IDestination) error {
	p.collection.Range(func(miner MinerScheduler) bool {
		p.log.Infof("changing pool to %s for minerID %s", dest.GetHost(), miner.GetID())

		miner.Allocate(100, dest)

		return true
	})

	return nil
}
