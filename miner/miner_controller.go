package miner

import (
	"context"
	"fmt"
	"net"
	"time"

	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
	"gitlab.com/TitanInd/hashrouter/tcpserver"
)

type MinerController struct {
	defaultDest interfaces.IDestination
	collection  interfaces.ICollection[MinerScheduler]

	minerVettingPeriod time.Duration
	poolMinDuration    time.Duration
	poolMaxDuration    time.Duration
	poolConnTimeout    time.Duration

	log        interfaces.ILogger
	logStratum bool
}

func NewMinerController(defaultDest interfaces.IDestination, collection interfaces.ICollection[MinerScheduler], log interfaces.ILogger, logStratum bool, minerVettingPeriod time.Duration, poolMinDuration, poolMaxDuration time.Duration, poolConnTimeout time.Duration) *MinerController {
	return &MinerController{
		defaultDest:        defaultDest,
		log:                log,
		collection:         collection,
		logStratum:         logStratum,
		minerVettingPeriod: minerVettingPeriod,
		poolMinDuration:    poolMinDuration,
		poolMaxDuration:    poolMaxDuration,
		poolConnTimeout:    poolConnTimeout,
	}
}

func (p *MinerController) HandleConnection(ctx context.Context, incomingConn net.Conn) error {
	buffered := tcpserver.NewBufferedConn(incomingConn)
	bytes, err := tcpserver.PeekNewLine(buffered)
	if err != nil {
		// connection is closed in the invoking function
		return err
	}

	m, err := stratumv1_message.ParseMessageToPool(bytes, p.log)
	if err != nil {
		return fmt.Errorf("invalid incoming message %s", string(bytes))
	}
	if _, ok := m.(*stratumv1_message.MiningUnknown); ok {
		return fmt.Errorf("invalid incoming message %s", string(bytes))
	}

	incomingConn = buffered

	p.log.Warnf("Miner connected %s", incomingConn.RemoteAddr())

	logMiner := p.log.Named(incomingConn.RemoteAddr().String())

	poolPool := protocol.NewStratumV1PoolPool(logMiner, p.poolConnTimeout, p.logStratum)
	err = poolPool.SetDest(context.TODO(), p.defaultDest, nil)
	if err != nil {
		p.log.Error(err)
		return err
	}
	extranonce, size := poolPool.GetExtranonce()
	msg := stratumv1_message.NewMiningSubscribeResult(extranonce, size)
	miner := protocol.NewStratumV1MinerConn(incomingConn, logMiner, msg, p.logStratum, time.Now())
	validator := hashrate.NewHashrate(logMiner)
	minerModel := protocol.NewStratumV1MinerModel(poolPool, miner, validator, logMiner)

	destSplit := NewDestSplit()

	minerScheduler := NewOnDemandMinerScheduler(minerModel, destSplit, logMiner, p.defaultDest, p.minerVettingPeriod, p.poolMinDuration, p.poolMaxDuration)

	p.collection.Store(minerScheduler)

	err = minerScheduler.Run(ctx)

	p.log.Warnf("Miner disconnected %s %s", incomingConn.RemoteAddr(), err)
	p.collection.Delete(minerScheduler.GetID())

	return err
}

func (p *MinerController) ChangeDestAll(dest interfaces.IDestination) error {
	p.collection.Range(func(miner MinerScheduler) bool {
		p.log.Infof("changing pool to %s for minerID %s", dest.GetHost(), miner.GetID())

		split := NewDestSplit()
		split, _ = split.Allocate("API_TEST", 1, dest, nil)
		miner.SetDestSplit(split)

		return true
	})

	return nil
}
