package miner

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
)

// DefaultDestID is used as destinationID / contractID for split serving default pool
const DefaultDestID = "default-dest"
const DestHistorySize = 2048

// OnDemandMinerScheduler distributes the hashpower of a single miner in time for multiple destinations
type OnDemandMinerScheduler struct {
	minerModel MinerModel
	log        interfaces.ILogger

	destSplit         *DestSplit // current hashrate distribution without default pool, watch also getDestSplitWithDefault()
	upcomingDestSplit *DestSplit // DestSplit that will be enabled on the next destination cycle

	history          *DestHistory // destination history log
	lastDestChangeAt time.Time
	restartDestCycle chan struct{} // discards current dest item and starts over the cycle

	defaultDest        interfaces.IDestination // the default destination that is used for unallocated part of destSplit
	minerVettingPeriod time.Duration           // duration during which miner wont fulfill any contract (its state is vetting) right after its connection
	destMinUptime      time.Duration           // minimum time miner allowed to be pointed to the destination pool to get some job done
	destMaxDowntime    time.Duration           // maximum time miner allowed not to provide submits to the destination pool to be considered online
}

func NewOnDemandMinerScheduler(minerModel MinerModel, destSplit *DestSplit, log interfaces.ILogger, defaultDest interfaces.IDestination, minerVettingPeriod, destMinUptime, destMaxDowntime time.Duration) *OnDemandMinerScheduler {
	history := NewDestHistory(DestHistorySize)
	history.Add(defaultDest, DefaultDestID, nil)

	scheduler := &OnDemandMinerScheduler{
		minerModel:         minerModel,
		destSplit:          destSplit,
		log:                log,
		defaultDest:        defaultDest,
		minerVettingPeriod: minerVettingPeriod,
		destMinUptime:      destMinUptime,
		destMaxDowntime:    destMaxDowntime,
		history:            history,
		restartDestCycle:   make(chan struct{}, 1),
	}

	minerModel.OnFault(scheduler.onFault)

	return scheduler
}

func (m *OnDemandMinerScheduler) Run(ctx context.Context) error {
	minerModelErr := make(chan error)
	go func() {
		minerModelErr <- m.minerModel.Run(ctx)
	}()

	for {
		destinations := m.getUpdatedDestSplitWithDefault()

	DEST_CYCLE:
		for _, splitItem := range destinations.Iter() {
			m.log.Debugf("new dest cycle: old dest %s, upcoming dest %s", m.minerModel.GetDest(), splitItem.Dest)

			if !lib.IsEqualDest(m.minerModel.GetDest(), splitItem.Dest) {
				m.log.Debugf("changing dest to %s", splitItem.Dest)

				err := m.ChangeDest(context.TODO(), splitItem.Dest, splitItem.ID, splitItem.OnSubmit)
				if err != nil {
					// if change dest fails then it is likely something wrong with the dest pool
					m.log.Errorf("cannot change dest to %s %s", splitItem.Dest, err)
					m.log.Warnf("falling back to default dest for current split item")
					err := m.ChangeDest(context.TODO(), m.defaultDest, splitItem.ID, splitItem.OnSubmit)
					if err != nil {
						return err
					}
				}
				m.log.Debugf("miner %s dest changed to %s", m.minerModel.GetID(), m.minerModel.GetDest())
				m.lastDestChangeAt = time.Now()
			}

			splitDuration := time.Duration(float64(m.getCycleDuration()) * splitItem.Fraction)
			m.log.Infof("destination %s for %.2f seconds", splitItem.Dest, splitDuration.Seconds())

			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-minerModelErr:
				return err
			case <-m.restartDestCycle:
				m.log.Infof("destination cycle restarted")
				break DEST_CYCLE
			case <-time.After(splitDuration):
			}
		}
	}
}

func (m *OnDemandMinerScheduler) getCycleDuration() time.Duration {
	return m.destMinUptime + m.destMaxDowntime
}

func (m *OnDemandMinerScheduler) GetID() string {
	return m.minerModel.GetID()
}

// GetUnallocatedHashrate returns the available miner hashrate
func (m *OnDemandMinerScheduler) GetUnallocatedHashrateGHS() int {
	// the remainder should be small enough to ignore
	return int(m.destSplit.GetUnallocated() * float64(m.minerModel.GetHashRateGHS()))
}

// GetDestSplit Returns current or upcoming destination split if available
func (m *OnDemandMinerScheduler) GetCurrentDestSplit() *DestSplit {
	return m.destSplit.Copy()
}

// GetDestSplit Returns current or upcoming destination split if available
func (m *OnDemandMinerScheduler) GetDestSplit() *DestSplit {
	if m.upcomingDestSplit != nil {
		return m.upcomingDestSplit.Copy()
	}
	return m.destSplit.Copy()
}

func (m *OnDemandMinerScheduler) GetUpcomingDestSplit() *DestSplit {
	if m.upcomingDestSplit != nil {
		return m.upcomingDestSplit.Copy()
	}
	return nil
}

// SetDestSplit sets upcoming destination split which will be used on next cycle
func (m *OnDemandMinerScheduler) SetDestSplit(upcomingDestSplit *DestSplit) {
	shouldRestartDestCycle := false

	if m.destSplit.IsEmpty() {
		shouldRestartDestCycle = true
	} else if upcomingDestSplit.IsEmpty() {
		shouldRestartDestCycle = true
	} else {
		if m.destSplit.Iter()[0].Fraction == 1 {
			shouldRestartDestCycle = true
		}
	}

	m.upcomingDestSplit = upcomingDestSplit.Copy()
	if shouldRestartDestCycle {
		m.restartDestCycle <- struct{}{}
	}

	m.log.Infof("new destination split: %s", upcomingDestSplit.String())
}

// ChangeDest forcefully change destination regardless of the split. The destination will be overrided back on next split item
func (m *OnDemandMinerScheduler) ChangeDest(ctx context.Context, dest interfaces.IDestination, ID string, onSubmit interfaces.IHashrate) error {
	m.history.Add(dest, ID, nil)
	return m.minerModel.ChangeDest(ctx, dest, onSubmit)
}

func (m *OnDemandMinerScheduler) GetHashRateGHS() int {
	return m.minerModel.GetHashRateGHS()
}

func (m *OnDemandMinerScheduler) GetHashRate() interfaces.Hashrate {
	return m.minerModel.GetHashRate()
}

// getUpdatedDestSplitWithDefault activates upcomingDestSplit and points remaining hashpower to default destination
func (m *OnDemandMinerScheduler) getUpdatedDestSplitWithDefault() *DestSplit {
	if m.upcomingDestSplit != nil {
		m.destSplit = m.upcomingDestSplit
		m.upcomingDestSplit = nil
	}

	return m.destSplit.AllocateRemaining(DefaultDestID, m.defaultDest, nil)
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

func (s *OnDemandMinerScheduler) IsVetting() bool {
	return time.Since(s.GetConnectedAt()) <= s.minerVettingPeriod
}

func (s *OnDemandMinerScheduler) IsFaulty() bool {
	return s.minerModel.IsFaulty()
}

func (s *OnDemandMinerScheduler) GetStatus() MinerStatus {
	if s.IsVetting() {
		return MinerStatusVetting
	}
	if s.destSplit.IsEmpty() {
		return MinerStatusFree
	}
	return MinerStatusBusy
}

func (s *OnDemandMinerScheduler) RangeDestConn(f func(key any, value any) bool) {
	s.minerModel.RangeDestConn(f)
}

func (s *OnDemandMinerScheduler) RangeHistory(f func(item HistoryItem) bool) {
	s.history.Range(f)
}

func (s *OnDemandMinerScheduler) RangeHistoryContractID(contractID string, f func(item HistoryItem) bool) {
	s.history.RangeContractID(contractID, f)
}

func (s *OnDemandMinerScheduler) onFault(ctx context.Context) {
	s.SetDestSplit(NewDestSplit())
}

var _ MinerScheduler = new(OnDemandMinerScheduler)
