package miner

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

// OnDemandMinerScheduler is responsible for distributing the resources of a single miner across multiple destinations
// and falling back to default pool for unallocated resources
type OnDemandMinerScheduler struct {
	minerModel MinerModel
	destSplit  *DestSplit
	reset      chan struct{} // used to start over the destination cycle after update has been made
	log        interfaces.ILogger
}

// const ON_DEMAND_SWITCH_TIMEOUT = 10 * time.Minute
const ON_DEMAND_SWITCH_TIMEOUT = 1 * time.Minute

func NewOnDemandMinerScheduler(minerModel MinerModel, destSplit *DestSplit, log interfaces.ILogger) *OnDemandMinerScheduler {
	return &OnDemandMinerScheduler{
		minerModel,
		destSplit,
		make(chan struct{}),
		log,
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

	cycle:
		for _, splitItem := range m.destSplit.Iter() {
			m.log.Infof("changing destination to %s ", splitItem.DestAddr)

			err := m.minerModel.ChangeDest(splitItem.DestAddr, splitItem.DestUser, splitItem.DestPassword)
			if err != nil {
				return err
			}

			splitDuration := time.Duration(int64(ON_DEMAND_SWITCH_TIMEOUT/100) * int64(splitItem.Percentage))
			m.log.Infof("destination was changed to %s for %.2f seconds", splitItem.DestAddr, splitDuration.Seconds())

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

// GetUnallocatedHashpower returns the available miner hashrate
// TODO: discuss with a team. As hashpower may fluctuate, define some kind of expected hashpower being
// the average hashpower value excluding the periods potential drop during reconnection
func (m *OnDemandMinerScheduler) GetUnallocatedHashpower() int {
	// the remainder should be small enough to ignore
	return int(m.destSplit.GetUnallocated()) * m.minerModel.GetHashRate() / 100
}

// IsBusy returns true if miner is fulfilling at least one contract
func (m *OnDemandMinerScheduler) IsBusy() bool {
	return m.destSplit.GetAllocated() > 0
}

func (m *OnDemandMinerScheduler) SetDestSplit(destSplit *DestSplit) {
	m.destSplit = destSplit
}

// Allocate directs miner resources to the destination
func (m *OnDemandMinerScheduler) Allocate(percentage uint8, destAddr, destUser, destPassword string) {
	m.destSplit.Allocate(percentage, destAddr, destUser, destPassword)
}

// Deallocate removes destination from miner's resource allocation
func (m *OnDemandMinerScheduler) Deallocate(percentage uint8, destAddr, destUser, destPassword string) (ok bool) {
	return m.destSplit.Deallocate(destAddr, destUser)
}

func (m *OnDemandMinerScheduler) GetHashRate() int {
	return m.minerModel.GetHashRate()
}
