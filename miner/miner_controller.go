package miner

import (
	"context"
	"fmt"
	"net"
	"time"

	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
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

	submitErrLimit int

	globalSubmitTracker interfaces.GlobalHashrate

	log         interfaces.ILogger
	logProtocol bool
}

func NewMinerController(
	defaultDest interfaces.IDestination,
	collection interfaces.ICollection[MinerScheduler],
	log interfaces.ILogger,
	logProtocol bool,
	minerVettingPeriod, poolMinDuration, poolMaxDuration, poolConnTimeout time.Duration,
	submitErrLimit int,
	globalSubmitTracker interfaces.GlobalHashrate,
) *MinerController {
	return &MinerController{
		defaultDest:         defaultDest,
		log:                 log,
		collection:          collection,
		logProtocol:         logProtocol,
		minerVettingPeriod:  minerVettingPeriod,
		poolMinDuration:     poolMinDuration,
		poolMaxDuration:     poolMaxDuration,
		poolConnTimeout:     poolConnTimeout,
		submitErrLimit:      submitErrLimit,
		globalSubmitTracker: globalSubmitTracker,
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

	p.log.Warnf("Miner connected %s->%s", incomingConn.RemoteAddr(), incomingConn.LocalAddr())

	logMiner := p.log.Named(incomingConn.RemoteAddr().String())

	poolPool := protocol.NewStratumV1PoolPool(logMiner, p.poolConnTimeout, incomingConn.RemoteAddr().String(), p.logProtocol)
	err = poolPool.SetDest(context.TODO(), p.defaultDest, nil)
	if err != nil {
		p.log.Error(err)
		return err
	}
	extranonce, size := poolPool.GetExtranonce()
	msg := stratumv1_message.NewMiningSubscribeResult(extranonce, size)

	var protocolLog interfaces.ILogger = nil
	if p.logProtocol {
		protocolLog, err = lib.NewFileLogger("MINER-" + incomingConn.RemoteAddr().String())
		if err != nil {
			p.log.Error(err)
		}
	}

	miner := protocol.NewStratumV1MinerConn(incomingConn, msg, time.Now(), p.poolConnTimeout, logMiner, protocolLog)
	validator := hashrate.NewHashrateV2(hashrate.NewSma(p.poolMinDuration + p.poolMaxDuration))
	minerModel := protocol.NewStratumV1MinerModel(poolPool, miner, validator, p.submitErrLimit, p.globalSubmitTracker, logMiner)

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
