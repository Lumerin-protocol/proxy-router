package contractmanager

import (
	"context"
	"fmt"
	"math"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/miner"
)

type GlobalSchedulerV2 struct {
	// configuration
	poolMinDuration       time.Duration
	poolMaxDuration       time.Duration
	hashrateDiffThreshold float64

	// dependencies
	minerCollection interfaces.ICollection[miner.MinerScheduler]
	log             interfaces.ILogger

	// internal vars
	poolMinFraction float64
	poolMaxFraction float64
	queue           chan task
}

type task struct {
	contractID  string
	hashrateGHS int
	dest        interfaces.IDestination
	errCh       chan error // nil for success, err for error
}

const (
	TASK_PUSH_ALERT_TIMEOUT = 10 * time.Second
	TASK_ALERT_TIMEOUT      = 10 * time.Second
)

func NewGlobalSchedulerV2(minerCollection interfaces.ICollection[miner.MinerScheduler], log interfaces.ILogger, poolMinDuration, poolMaxDuration time.Duration, hashrateDiffThreshold float64) *GlobalSchedulerV2 {
	instance := &GlobalSchedulerV2{
		minerCollection:       minerCollection,
		log:                   log,
		hashrateDiffThreshold: hashrateDiffThreshold,
		queue:                 make(chan task, 100),
	}
	instance.setPoolDurationConstraints(poolMinDuration, poolMaxDuration)
	return instance
}

func (s *GlobalSchedulerV2) Run(ctx context.Context) error {
	for {
		select {
		case tsk := <-s.queue:
			s.fulfillTask(ctx, tsk)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *GlobalSchedulerV2) GetMinerSnapshot() *AllocSnap {
	snap := CreateMinerSnapshot(s.minerCollection)
	return &snap
}

// Update publishes adjusts contract hashrate task. Set hashrateGHS to 0 to deallocate miners.
func (s *GlobalSchedulerV2) Update(contractID string, hashrateGHS int, dest interfaces.IDestination) error {
	errCh := make(chan error)
	tsk := task{
		contractID:  contractID,
		hashrateGHS: hashrateGHS,
		dest:        dest,
		errCh:       errCh,
	}

	s.pushTask(tsk)

	return s.waitTask(tsk)
}

// update checks contract hashrate and updates miner allocation if it is outside of s.hashrateDiffThreshold
func (s *GlobalSchedulerV2) update(contractID string, targetHrGHS int, dest interfaces.IDestination) error {
	s.log.Debugf("received task contractID(%s), hashrate(%d), dest(%s)", lib.AddrShort(contractID), targetHrGHS, dest)

	snap := s.GetMinerSnapshot()
	currentHrGHS := 0
	miners, ok := snap.Contract(contractID)
	if !ok {
		miners = NewAllocCollection()
	}

	if ok {
		s.log.Debugf("allocation for contractID(%s) before: %s", lib.AddrShort(contractID), miners.String())
		currentHrGHS = miners.GetAllocatedGHS()
	}

	// deallocate totally
	if targetHrGHS == 0 {
		s.deallocate(miners)
		s.log.Debugf("contractID(%s) totally deallocated", lib.AddrShort(contractID))
		return nil
	}

	var (
		allocCollection *AllocCollection
		err             error
	)

	if lib.AlmostEqual(targetHrGHS, currentHrGHS, s.hashrateDiffThreshold) {
		s.log.Debugf("contractID(%s) targetGHS(%d) currentHrGHS(%d) is within diff threshold(%.2f)", lib.AddrShort(contractID), targetHrGHS, currentHrGHS, s.hashrateDiffThreshold)
		allocCollection = miners
	} else if targetHrGHS > currentHrGHS {
		allocCollection, err = s.increaseHr(snap, targetHrGHS-currentHrGHS, contractID, dest)
	} else {
		allocCollection, err = s.decreaseHr(snap, currentHrGHS-targetHrGHS, contractID, dest)
	}
	if err != nil {
		s.log.Error("allocation error, no changes has been made %s", err)
		return nil
	}

	allocCollection = s.adjustAllocCollection(allocCollection)
	s.log.Debugf("allocation for contractID(%s) after: %s", lib.AddrShort(contractID), allocCollection.String())

	return s.applyAllocCollection(contractID, allocCollection, dest)
}

func (s *GlobalSchedulerV2) increaseHr(snap *AllocSnap, hrToIncreaseGHS int, contractID string, dest interfaces.IDestination) (*AllocCollection, error) {
	s.log.Debugf("increasing allocation contractID(%s) hrToIncrease(%d)", lib.AddrShort(contractID), hrToIncreaseGHS)

	// 1. check if existing miners can be used to increase hashrate
	remainingToAddGHS := hrToIncreaseGHS
	newContractItems := NewAllocCollection()

	allocItems, ok := snap.Contract(contractID)
	if ok {
		// add existing items into new contract items and max out their percentage
		for _, item := range allocItems.GetItems() {
			if remainingToAddGHS == 0 {
				break
			}
			minerID := item.MinerID
			minerItems, ok := snap.Miner(minerID)
			if !ok {
				s.log.DPanicf("miner not found")
			}
			_, available := minerItems.GetUnallocatedGHS()
			toAddGHS := lib.MinInt(remainingToAddGHS, available.AllocatedGHS())
			toAddFraction := float64(toAddGHS) / float64(item.TotalGHS)

			newContractItems.Add(item.MinerID, &AllocItem{
				ContractID: contractID,
				MinerID:    minerID,
				Fraction:   item.Fraction + toAddFraction, // it can fall into unavailable interval, will be adjusted later
			})
			remainingToAddGHS -= toAddGHS
		}
	}

	// 2. find additional miners that can fulfill the contract
	if remainingToAddGHS > 0 {
		_, minerHashrates := snap.GetUnallocatedGHS()

		combination, delta := FindCombinations(minerHashrates.FilterFullyAvailable(), remainingToAddGHS)

		if combination.Len() == 0 {
			return nil, fmt.Errorf("cannot fulfill given hashrate")
		}

		// safety cycle to remove only from alloc item with available hashrate
		for _, ai := range combination.SortByAllocatedGHS() {
			if delta <= ai.TotalGHS {
				fractionToRemove := float64(delta) / float64(ai.TotalGHS)
				item, _ := combination.Get(ai.GetSourceID())
				item.Fraction = ai.Fraction - fractionToRemove
				break
			}
		}

		for _, ai := range combination.GetItems() {
			newContractItems.Add(contractID, ai)
		}
	}

	//3. TODO: try to allocate from scratch (ignoring existing miners) and compare

	return newContractItems, nil
}

func (s *GlobalSchedulerV2) decreaseHr(snap *AllocSnap, hrToDecreaseGHS int, contractID string, dest interfaces.IDestination) (*AllocCollection, error) {
	remainingToRemoveGHS := hrToDecreaseGHS
	newContractItems := NewAllocCollection()
	allocItems, ok := snap.Contract(contractID)
	if !ok {
		s.log.DPanicf("contract(%s) not found", lib.AddrShort(contractID))
	}

	// 1. use existing miners to decrease hashrate
	for _, item := range allocItems.SortByAllocatedGHS() {
		if remainingToRemoveGHS == 0 {
			break
		}

		toRemoveGHS := lib.MinInt(remainingToRemoveGHS, item.AllocatedGHS())

		toRemoveFraction := float64(toRemoveGHS) / float64(item.TotalGHS)
		if toRemoveGHS == item.AllocatedGHS() {
			toRemoveFraction = item.Fraction // avoid float comparison error
		}

		newContractItems.Add(item.MinerID, &AllocItem{
			ContractID: contractID,
			MinerID:    item.MinerID,
			Fraction:   item.Fraction - toRemoveFraction, // it can fall into unavailable interval, will be adjusted later, zero means deallocate
			TotalGHS:   item.TotalGHS,
		})
		remainingToRemoveGHS -= toRemoveGHS
	}

	if remainingToRemoveGHS != 0 {
		s.log.Warnf("inconsistensy error, shouldnt go here")
	}

	newContractItems = s.adjustAllocCollection(newContractItems)

	//5. Apply updated rules
	return newContractItems, nil
}

// adjustAllocCollection adjusts percentage for each allocation item so it wont fall in red zone
func (s *GlobalSchedulerV2) adjustAllocCollection(coll *AllocCollection) *AllocCollection {
	// TODO
	// 0. check is alloc collection violates constraints

	// 1. check if split items could be merged together with constraints

	// 2. apply constraints by adding one more miner or reducing existing miners

	// 3. attempt to drop everything and allocate from scratch and compare (but try to avoid unnecessary reallocations)
	return coll
}

func (s *GlobalSchedulerV2) applyAllocCollection(contractID string, coll *AllocCollection, dest interfaces.IDestination) error {
	for _, item := range coll.GetItems() {
		miner, ok := s.minerCollection.Load(item.GetSourceID())
		if !ok {
			s.log.Warnf("unknown miner: %v, skipping", item.GetSourceID())
			continue
		}

		if item.Fraction == 0 {
			destSplit, ok := miner.GetDestSplit().RemoveByID(contractID)
			if ok {
				miner.SetDestSplit(destSplit)
			}
			continue
		}

		destSplitItem, ok := miner.GetDestSplit().GetByID(contractID)
		isNotChanged := ok && destSplitItem.Dest.IsEqual(dest) && destSplitItem.Fraction == item.Fraction

		if isNotChanged {
			s.log.Debugf("miners update skipped due to no changes")
		} else {
			destSplit, err := miner.GetDestSplit().UpsertFractionByID(contractID, item.Fraction, dest)
			if err != nil {
				return err
			}
			miner.SetDestSplit(destSplit)
		}
	}

	return nil
}

func (s *GlobalSchedulerV2) deallocate(coll *AllocCollection) {
	for _, item := range coll.GetItems() {
		miner, ok := s.minerCollection.Load(item.GetSourceID())
		if !ok {
			s.log.Warnf("unknown miner: %s, skipping", item.GetSourceID())
			continue
		}

		destSplit, ok := miner.GetDestSplit().RemoveByID(item.ContractID)
		if !ok {
			s.log.Warnf("unknown split: %s, skipping", item.GetSourceID())
			continue
		}
		miner.SetDestSplit(destSplit)
	}
}

func (s *GlobalSchedulerV2) setPoolDurationConstraints(min, max time.Duration) {
	s.poolMinDuration, s.poolMaxDuration = min, max
	s.poolMinFraction = float64(min) / float64(min+max)
	s.poolMaxFraction = float64(max) / float64(min+max)
}

// pushTask warns if bufferised channel is full and goroutine is blocked
func (s *GlobalSchedulerV2) pushTask(tsk task) {
	var t time.Duration
	for t = 0; ; t += TASK_PUSH_ALERT_TIMEOUT {
		select {
		case s.queue <- tsk:
			return
		case <-time.After(TASK_PUSH_ALERT_TIMEOUT):
			s.log.Warnf("ALERT task push takes too long: %s", t)
		}
	}
}

// waitTask warns if task is taking too long to fulfill
func (s *GlobalSchedulerV2) waitTask(tsk task) error {
	var t time.Duration
	for t = 0; ; t += TASK_ALERT_TIMEOUT {
		select {
		case err := <-tsk.errCh:
			return err
		case <-time.After(TASK_ALERT_TIMEOUT):
			s.log.Warnf("ALERT task takes too long: %s", t)
		}
	}
}

func (s *GlobalSchedulerV2) fulfillTask(ctx context.Context, tsk task) {
	tsk.errCh <- s.update(tsk.contractID, tsk.hashrateGHS, tsk.dest)
}

func (s *GlobalSchedulerV2) IsDeliveringAdequateHashrate(ctx context.Context, targetHashrateGHS int, dest interfaces.IDestination, hashrateDiffThreshold float64) bool {
	var actualHashrate int

	s.minerCollection.Range(func(miner miner.MinerScheduler) bool {
		if miner.GetWorkerName() == dest.Username() {
			actualHashrate += miner.GetHashRateGHS()
		}
		return true
	})

	deltaGHS := targetHashrateGHS - actualHashrate
	s.log.Debugf("target hashrate %d, actual hashrate %d, delta %d", targetHashrateGHS, actualHashrate, deltaGHS)

	if deltaGHS < 0 || math.Abs(float64(deltaGHS))/float64(targetHashrateGHS) < hashrateDiffThreshold {
		s.log.Debugf("contract delivering enough hashrate")
		return true
	}

	return false
}
