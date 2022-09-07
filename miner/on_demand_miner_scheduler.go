package miner

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/protocol"
)

// OnDemandMinerScheduler is responsible for distributing the resources of a single miner across multiple destinations
// and falling back to default pool for unallocated resources
type OnDemandMinerScheduler struct {
	minerModel  MinerModel
	destSplit   *DestSplit    // may be not allocated fully, the remaining will be directed to defaultDest
	reset       chan struct{} // used to start over the destination cycle after update has been made
	log         interfaces.ILogger
	defaultDest interfaces.IDestination // the default destination that is used for unallocated part of destSplit
}

// const ON_DEMAND_SWITCH_TIMEOUT = 10 * time.Minute
const ON_DEMAND_SWITCH_TIMEOUT = 30 * time.Minute

func NewOnDemandMinerScheduler(minerModel MinerModel, destSplit *DestSplit, log interfaces.ILogger, defaultDest interfaces.IDestination) *OnDemandMinerScheduler {
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
		if len(m.getDest().Iter()) == 1 {
			splitItem := m.getDest().Iter()[0]
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
		// TODO: generalize cycle function to be used by both single and multiple destinations
	cycle:
		for _, splitItem := range m.getDest().Iter() {
			m.log.Infof("changing destination to %s", splitItem.Dest)

			err := m.minerModel.ChangeDest(splitItem.Dest)
			if err != nil {
				return err
			}

			splitDuration := time.Duration(int64(ON_DEMAND_SWITCH_TIMEOUT/100) * int64(splitItem.Percentage))
			m.log.Infof("destination was changed to %s for %.2f seconds", splitItem.Dest, splitDuration.Seconds())

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
func (m *OnDemandMinerScheduler) GetUnallocatedHashrateGHS() int {
	// the remainder should be small enough to ignore
	return int(m.destSplit.GetUnallocated()) * m.minerModel.GetHashRateGHS() / 100
}

// IsBusy returns true if miner is fulfilling at least one contract
func (m *OnDemandMinerScheduler) IsBusy() bool {
	return m.destSplit.GetAllocated() > 0
}

func (m *OnDemandMinerScheduler) SetDestSplit(destSplit *DestSplit) {
	m.destSplit = destSplit
}

func (m *OnDemandMinerScheduler) GetDestSplit() *DestSplit {
	return m.destSplit
}

// Allocate directs miner resources to the destination
func (m *OnDemandMinerScheduler) Allocate(percentage float64, dest interfaces.IDestination) (*Split, error) {
	defer m.resetDestCycle()
	return m.destSplit.Allocate(percentage, dest)
}

// ChangeDest forcefully change destination
//  may cause issues when split is enabled
func (m *OnDemandMinerScheduler) ChangeDest(dest lib.Dest) error {
	return m.minerModel.ChangeDest(dest)
}

func (m *OnDemandMinerScheduler) GetHashRateGHS() int {
	return m.minerModel.GetHashRateGHS()
}

// getDest adds default destination to remaining part of destination split
func (m *OnDemandMinerScheduler) getDest() *DestSplit {
	dest := m.destSplit.Copy()
	dest.AllocateRemaining(m.defaultDest)
	return dest
}

func (m *OnDemandMinerScheduler) OnSubmit(cb protocol.OnSubmitHandler) protocol.ListenerHandle {
	return m.minerModel.OnSubmit(cb)
}

func (m *OnDemandMinerScheduler) GetCurrentDest() interfaces.IDestination {
	return m.minerModel.GetDest()
}

func (m *OnDemandMinerScheduler) GetCurrentDifficulty() int {
	return m.minerModel.GetCurrentDifficulty()
}

func (m *OnDemandMinerScheduler) GetWorkerName() string {
	return m.minerModel.GetWorkerName()
}

// resetDestCycle signals that destSplit has been changed, and starts new destination cycle
func (m *OnDemandMinerScheduler) resetDestCycle() {
	m.reset <- struct{}{}
}
