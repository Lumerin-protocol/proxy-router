package miner

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol"
)

const DefaultDestID = "default-dest"

// OnDemandMinerScheduler is responsible for distributing the resources of a single miner across multiple destinations
// and falling back to default pool for unallocated resources
type OnDemandMinerScheduler struct {
	minerModel       MinerModel
	destSplit        *DestSplit // may be not allocated fully, the remaining will be directed to defaultDest
	log              interfaces.ILogger
	defaultDest      interfaces.IDestination // the default destination that is used for unallocated part of destSplit
	lastDestChangeAt time.Time

	minerVettingPeriod time.Duration
	destMinUptime      time.Duration
	destMaxUptime      time.Duration

	restartDestCycle chan struct{}
}

func NewOnDemandMinerScheduler(minerModel MinerModel, destSplit *DestSplit, log interfaces.ILogger, defaultDest interfaces.IDestination, minerVettingPeriod, destMinUptime, destMaxUptime time.Duration) *OnDemandMinerScheduler {
	return &OnDemandMinerScheduler{
		minerModel:       minerModel,
		destSplit:        destSplit,
		log:              log,
		defaultDest:      defaultDest,
		destMinUptime:    destMinUptime,
		destMaxUptime:    destMaxUptime,
		restartDestCycle: make(chan struct{}, 1),
	}
}

func (m *OnDemandMinerScheduler) Run(ctx context.Context) error {
	minerModelErr := make(chan error)
	go m.minerModel.Run(ctx, minerModelErr)

	for {
		destinations := m.getDest().Iter()
		m.log.Infof("new scheduler cycle for miner(%s), destinations(%d)", m.GetID(), len(destinations))

	DEST_CYCLE:
		for _, splitItem := range destinations {
			if !m.minerModel.GetDest().IsEqual(splitItem.Dest) {
				m.log.Infof("changing destination to %s", splitItem.Dest)
				err := m.minerModel.ChangeDest(splitItem.Dest)
				if err != nil {
					return err
				}
				m.lastDestChangeAt = time.Now()
			}

			splitDuration := time.Duration(float64(m.getCycleDuration()) * splitItem.Percentage)
			m.log.Infof("destination %s for %.2f seconds", splitItem.Dest, splitDuration.Seconds())

			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-minerModelErr:
				return err
			case <-m.restartDestCycle:
				break DEST_CYCLE
			case <-time.After(splitDuration):
			}
		}
	}
}

func (m *OnDemandMinerScheduler) getCycleDuration() time.Duration {
	return m.destMinUptime + m.destMaxUptime
}

func (m *OnDemandMinerScheduler) GetID() string {
	return m.minerModel.GetID()
}

// GetUnallocatedPercentage returns the percentage of power of a miner available to fulfill some contact
func (m *OnDemandMinerScheduler) GetUnallocatedPercentage() float64 {
	return m.destSplit.GetUnallocated()
}

// GetUnallocatedHashrate returns the available miner hashrate
// TODO: discuss with a team. As hashpower may fluctuate, define some kind of expected hashpower being
// the average hashpower value excluding the periods potential drop during reconnection
func (m *OnDemandMinerScheduler) GetUnallocatedHashrateGHS() int {
	// the remainder should be small enough to ignore
	return int(m.destSplit.GetUnallocated() * float64(m.minerModel.GetHashRateGHS()))
}

func (m *OnDemandMinerScheduler) GetDestSplit() *DestSplit {
	return m.destSplit
}

// Allocate directs miner resources to the destination
func (m *OnDemandMinerScheduler) Allocate(ID string, percentage float64, dest interfaces.IDestination) (*Split, error) {
	oldDestSplit := m.destSplit.Copy()
	split, err := m.destSplit.Allocate(ID, percentage, dest)
	if err != nil {
		return nil, err
	}
	// if miner was pointing only to default pool
	if len(oldDestSplit.Iter()) == 0 {
		m.restartDestCycle <- struct{}{}
	}
	return split, nil
}

func (m *OnDemandMinerScheduler) Deallocate(ID string) (ok bool) {
	ok = false

	for i, item := range m.destSplit.Iter() {
		if item.ID == ID {
			newSplit := append(m.destSplit.split[:i], m.destSplit.split[i+1:]...)
			m.destSplit.mutex.Lock()
			m.destSplit.split = newSplit
			m.destSplit.mutex.Unlock()

			ok = true
		}
	}

	return ok
}

// ChangeDest forcefully change destination
// may cause issues when split is enabled
func (m *OnDemandMinerScheduler) ChangeDest(dest interfaces.IDestination) error {
	return m.minerModel.ChangeDest(dest)
}

func (m *OnDemandMinerScheduler) GetHashRateGHS() int {
	return m.minerModel.GetHashRateGHS()
}

func (m *OnDemandMinerScheduler) GetHashRate() protocol.Hashrate {
	return m.minerModel.GetHashRate()
}

// getDest adds default destination to remaining part of destination split
func (m *OnDemandMinerScheduler) getDest() *DestSplit {
	dest := m.destSplit.Copy()
	dest.AllocateRemaining(DefaultDestID, m.defaultDest)
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

func (s *OnDemandMinerScheduler) GetConnectedAt() time.Time {
	return s.minerModel.GetConnectedAt()
}

func (s *OnDemandMinerScheduler) GetUptime() time.Duration {
	return time.Since(s.GetConnectedAt())
}

func (s *OnDemandMinerScheduler) IsVetted() bool {
	return time.Since(s.GetConnectedAt()) >= s.minerVettingPeriod
}

func (s *OnDemandMinerScheduler) GetStatus() MinerStatus {
	if !s.IsVetted() {
		return MinerStatusVetting
	}
	if len(s.destSplit.split) == 0 {
		return MinerStatusFree
	}
	return MinerStatusBusy
}

var _ MinerScheduler = new(OnDemandMinerScheduler)
