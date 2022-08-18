package miner

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
)

// OnDemandMinerScheduler is responsible for distributing the resources of a single miner across multiple destinations
// and falling back to default pool for unallocated resources
type OnDemandMinerScheduler struct {
	minerModel  MinerModel
	destSplit   *DestSplit    // may be not allocated fully, the remaining will be directed to defaultDest
	reset       chan struct{} // used to start over the destination cycle after update has been made
	log         interfaces.ILogger
	defaultDest interop.Dest // the default destination that is used for unallocated part of destSplit
}

// const ON_DEMAND_SWITCH_TIMEOUT = 10 * time.Minute
const ON_DEMAND_SWITCH_TIMEOUT = 1 * time.Minute

func NewOnDemandMinerScheduler(minerModel MinerModel, destSplit *DestSplit, log interfaces.ILogger, defaultDest interop.Dest) *OnDemandMinerScheduler {
	return &OnDemandMinerScheduler{
		minerModel,
		destSplit,
		make(chan struct{}),
		log,
		defaultDest,
	}
}

func (m *OnDemandMinerScheduler) Run(ctx context.Context) error {
	var minerModelErr chan error
	go func() {
		minerModelErr <- m.minerModel.Run()
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-minerModelErr:
			return err
		default:
		}

		// if only one destination
		if len(m.destSplit.Iter()) == 1 {
			splitItem := m.destSplit.Iter()[0]
			err := m.minerModel.ChangeDest(splitItem.Dest)
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-minerModelErr:
				return err
			case <-m.reset:
				continue
			}
		}

		// if multiple destinations
	cycle:
		for _, splitItem := range m.getDest().Iter() {
			m.log.Infof("changing destination to %s ", splitItem.Dest.Host)

			err := m.minerModel.ChangeDest(splitItem.Dest)
			if err != nil {
				return err
			}

			splitDuration := time.Duration(int64(ON_DEMAND_SWITCH_TIMEOUT/100) * int64(splitItem.Percentage))
			m.log.Infof("destination was changed to %s for %.2f seconds", splitItem.Dest.Host, splitDuration.Seconds())

			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-minerModelErr:
				return err
			case <-time.After(splitDuration):
				continue
			case <-m.reset:
				break cycle
			}
		}
	}
}

func (m *OnDemandMinerScheduler) GetID() string {
	return m.minerModel.GetID()
}

// GetUnallocatedPercentage returns the percentage of power of a miner available to fulfill some contact
func (m *OnDemandMinerScheduler) GetUnallocatedPercentage() uint8 {
	return m.destSplit.GetUnallocated()
}

// GetUnallocatedHashrate returns the available miner hashrate
// TODO: discuss with a team. As hashpower may fluctuate, define some kind of expected hashpower being
// the average hashpower value excluding the periods potential drop during reconnection
func (m *OnDemandMinerScheduler) GetUnallocatedHashrate() uint64 {
	// the remainder should be small enough to ignore
	return uint64(m.destSplit.GetUnallocated()) * m.minerModel.GetHashRate() / 100
}

// IsBusy returns true if miner is fulfilling at least one contract
func (m *OnDemandMinerScheduler) IsBusy() bool {
	return m.destSplit.GetAllocated() > 0
}

func (m *OnDemandMinerScheduler) SetDestSplit(destSplit *DestSplit) {
	m.destSplit = destSplit
}

// Allocate directs miner resources to the destination
func (m *OnDemandMinerScheduler) Allocate(percentage float64, dest interop.Dest) error {
	return m.destSplit.Allocate(percentage, dest)
}

// Deallocate removes destination from miner's resource allocation
func (m *OnDemandMinerScheduler) Deallocate(dest interop.Dest) (ok bool) {
	return m.destSplit.Deallocate(dest)
}

func (m *OnDemandMinerScheduler) GetHashRate() uint64 {
	return m.minerModel.GetHashRate()
}

// getDest adds default destination to remaining part of destination split
func (m *OnDemandMinerScheduler) getDest() *DestSplit {
	dest := m.destSplit.Copy()
	dest.AllocateRemaining(m.defaultDest)
	return dest
}
