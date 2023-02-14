package contractmanager

import (
	"context"
	"fmt"
	"math"
	"time"

	"gitlab.com/TitanInd/hashrouter/data"
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
	onSubmit    interfaces.IHashrate
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

// GetMinerSnapshot returns current or upcoming state of the miners is available
func (s *GlobalSchedulerV2) GetMinerSnapshot() *data.AllocSnap {
	snapshot := data.NewAllocSnap()

	s.minerCollection.Range(func(miner miner.MinerScheduler) bool {
		if miner.IsVetting() {
			return true
		}
		if miner.IsFaulty() {
			return true
		}

		hashrateGHS := miner.GetHashRateGHS()
		minerID := miner.GetID()

		snapshot.SetMiner(minerID, hashrateGHS)

		for _, splitItem := range miner.GetDestSplit().Iter() {
			snapshot.Set(minerID, splitItem.ID, splitItem.Fraction, hashrateGHS)
		}

		return true
	})

	return &snapshot
}

// Update publishes adjusts contract hashrate task. Set hashrateGHS to 0 to deallocate miners.
func (s *GlobalSchedulerV2) Update(contractID string, hashrateGHS int, dest interfaces.IDestination, onSubmit interfaces.IHashrate) error {
	errCh := make(chan error)
	tsk := task{
		contractID:  contractID,
		hashrateGHS: hashrateGHS,
		dest:        dest,
		errCh:       errCh,
		onSubmit:    onSubmit,
	}

	s.pushTask(tsk)

	return s.waitTask(tsk)
}

// update checks contract hashrate and updates miner allocation if it is outside of s.hashrateDiffThreshold
func (s *GlobalSchedulerV2) update(contractID string, targetHrGHS int, dest interfaces.IDestination, onSubmit interfaces.IHashrate) error {
	s.log.Debugf("received task contractID(%s), hashrate(%d), dest(%s)", lib.AddrShort(contractID), targetHrGHS, dest)

	snap := s.GetMinerSnapshot()
	currentHrGHS := 0
	miners, ok := snap.Contract(contractID)
	if !ok {
		miners = data.NewAllocCollection()
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
		allocCollection *data.AllocCollection
		isAccurate      bool = true
	)

	if lib.AlmostEqual(targetHrGHS, currentHrGHS, s.hashrateDiffThreshold) {
		s.log.Debugf("contractID(%s) targetGHS(%d) currentHrGHS(%d) is within diff threshold(%.2f)", lib.AddrShort(contractID), targetHrGHS, currentHrGHS, s.hashrateDiffThreshold)
		allocCollection = miners
	} else if targetHrGHS > currentHrGHS {
		allocCollection, isAccurate = s.increaseHr(snap, targetHrGHS-currentHrGHS, contractID, dest)
	} else {
		allocCollection, isAccurate = s.decreaseHr(snap, currentHrGHS-targetHrGHS, contractID, dest)
	}

	if !isAccurate {
		s.log.Warnf("allocation wasn't totally accurate, expected(%d), actual(%d)", targetHrGHS, allocCollection.GetAllocatedGHS())
	}

	s.log.Debugf("allocation for contractID(%s) before adjustment: %s", lib.AddrShort(contractID), allocCollection.String())
	allocCollection = s.adjustAllocCollection(allocCollection, snap)
	s.log.Debugf("allocation for contractID(%s) after adjustment: %s", lib.AddrShort(contractID), allocCollection.String())
	return s.applyAllocCollection(contractID, allocCollection, dest, onSubmit)
}

func (s *GlobalSchedulerV2) increaseHr(snap *data.AllocSnap, hrToIncreaseGHS int, contractID string, dest interfaces.IDestination) (coll *data.AllocCollection, isAccurate bool) {
	s.log.Debugf("increasing allocation contractID(%s) hrToIncrease(%d)", lib.AddrShort(contractID), hrToIncreaseGHS)
	remainingToAddGHS := hrToIncreaseGHS

	defer s.log.Debugf(
		"increased allocation for contractID(%s): actually added(%d), expected to add(%d)",
		lib.AddrShort(contractID), hrToIncreaseGHS-remainingToAddGHS, hrToIncreaseGHS,
	)

	// 1. check if existing miners can be used to increase hashrate
	allocItems, ok := snap.Contract(contractID)
	if ok {
		remainingToAddGHS = s.maxoutExisting(allocItems, remainingToAddGHS)
		if remainingToAddGHS == 0 {
			return allocItems, true
		}
	} else {
		allocItems = data.NewAllocCollection()
	}

	// 2. find additional miners that can fulfill the contract
	_, availableMinersHR := snap.GetUnallocatedGHS()
	remainingToAddGHS = s.addNewMiners(allocItems, availableMinersHR.FilterFullyAvailable(), remainingToAddGHS, contractID)
	if remainingToAddGHS == 0 {
		return allocItems, true
	}

	//3. TODO: try to allocate from scratch (ignoring existing miners) and compare

	return allocItems, false
}

// maxoutExisting tries to increase existing allocation up to 100%, returns GHS that is remaining
func (s *GlobalSchedulerV2) maxoutExisting(allocItems *data.AllocCollection, toAddGHS int) int {
	// adjust existing items to max out their percentage
	for _, item := range allocItems.GetItems() {
		if toAddGHS == 0 {
			return toAddGHS
		}

		availableGHS := int((1.0 - item.Fraction) * float64(item.TotalGHS))
		toAddGHSMiner := lib.MinInt(toAddGHS, availableGHS)
		toAddFraction := float64(toAddGHSMiner) / float64(item.TotalGHS)
		item.Fraction = item.Fraction + toAddFraction

		toAddGHS -= toAddGHSMiner
	}
	return toAddGHS
}

func (s *GlobalSchedulerV2) addNewMiners(allocItems *data.AllocCollection, freeMiners *data.AllocCollection, toAddGHS int, contractID string) (remainingGHS int) {
	combination, delta := FindCombinations(freeMiners, toAddGHS)

	if combination.Len() == 0 {
		return toAddGHS
	}

	overallocatedGHS := -delta

	if overallocatedGHS > 0 {
		// one of the miners should be partially allocated to account for delta
		for _, ai := range combination.SortByAllocatedGHS() {
			if overallocatedGHS <= ai.TotalGHS {
				fractionToRemove := float64(overallocatedGHS) / float64(ai.TotalGHS)
				item, _ := combination.Get(ai.GetSourceID())
				item.Fraction = ai.Fraction - fractionToRemove
				overallocatedGHS = 0
				break
			}
		}
	}

	s.log.Debugf("added miners: %s", combination.String())

	for _, ai := range combination.GetItems() {
		allocItems.Add(ai.MinerID, &data.AllocItem{
			MinerID:    ai.MinerID,
			ContractID: contractID,
			Fraction:   ai.Fraction,
			TotalGHS:   ai.TotalGHS,
		})
	}

	return -overallocatedGHS
}

func (s *GlobalSchedulerV2) decreaseHr(snap *data.AllocSnap, hrToDecreaseGHS int, contractID string, dest interfaces.IDestination) (coll *data.AllocCollection, isAccurate bool) {
	s.log.Debugf("decreasing allocation contractID(%s) hrToDecrease(%d)", lib.AddrShort(contractID), hrToDecreaseGHS)

	remainingToRemoveGHS := hrToDecreaseGHS
	newContractItems := data.NewAllocCollection()
	allocItems, ok := snap.Contract(contractID)
	if !ok {
		s.log.DPanicf("contract(%s) not found", lib.AddrShort(contractID))
	}

	// 1. use existing miners to decrease hashrate
	for _, item := range allocItems.SortByAllocatedGHS() {
		if remainingToRemoveGHS == 0 {
			// just add item into target collection without changes
			newContractItems.Add(item.MinerID, &data.AllocItem{
				ContractID: contractID,
				MinerID:    item.MinerID,
				Fraction:   item.Fraction,
				TotalGHS:   item.TotalGHS,
			})
			continue
		}

		toRemoveGHS := lib.MinInt(remainingToRemoveGHS, item.AllocatedGHS())

		toRemoveFraction := float64(toRemoveGHS) / float64(item.TotalGHS)
		if toRemoveGHS == item.AllocatedGHS() {
			toRemoveFraction = item.Fraction // avoid float comparison error
		}

		newContractItems.Add(item.MinerID, &data.AllocItem{
			ContractID: contractID,
			MinerID:    item.MinerID,
			Fraction:   item.Fraction - toRemoveFraction, // it can fall into unavailable interval, will be adjusted later, zero means deallocate
			TotalGHS:   item.TotalGHS,
		})

		remainingToRemoveGHS -= toRemoveGHS
	}

	if remainingToRemoveGHS != 0 {
		s.log.DPanicf("inconsistensy error, shouldnt go here")
	}

	newContractItems = s.adjustAllocCollection(newContractItems, snap)

	//5. Apply updated rules
	return newContractItems, true
}

func (s *GlobalSchedulerV2) checkRedZones(fraction float64) int {
	if fraction == 1 || fraction == 0 {
		return 0
	}
	if fraction < s.poolMinFraction {
		return -1
	}
	if fraction > s.poolMaxFraction {
		return +1
	}
	return 0
}

func (s *GlobalSchedulerV2) applyAllocCollection(contractID string, coll *data.AllocCollection, dest interfaces.IDestination, onSubmit interfaces.IHashrate) error {
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
		isNotChanged := ok && lib.IsEqualDest(destSplitItem.Dest, dest) && destSplitItem.Fraction == item.Fraction

		if isNotChanged {
			s.log.Debugf("miners update skipped due to no changes")
		} else {
			destSplit, err := miner.GetDestSplit().UpsertFractionByID(contractID, item.Fraction, dest, onSubmit)
			if err != nil {
				return err
			}
			miner.SetDestSplit(destSplit)
		}
	}

	return nil
}

func (s *GlobalSchedulerV2) deallocate(coll *data.AllocCollection) {
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
			close(tsk.errCh)
			return err
		case <-time.After(TASK_ALERT_TIMEOUT):
			s.log.Warnf("ALERT task takes too long: %s", t)
		}
	}
}

func (s *GlobalSchedulerV2) fulfillTask(ctx context.Context, tsk task) {
	err := s.update(tsk.contractID, tsk.hashrateGHS, tsk.dest, tsk.onSubmit)
	if err != nil {
		s.log.Error(err)
	}
	tsk.errCh <- err
}

func (s *GlobalSchedulerV2) IsDeliveringAdequateHashrate(ctx context.Context, targetHashrateGHS int, dest interfaces.IDestination, hashrateDiffThreshold float64) bool {
	var actualHashrate int

	s.minerCollection.Range(func(miner miner.MinerScheduler) bool {
		if miner.GetWorkerName() == dest.Username() {
			hr, ok := miner.GetHashRate().GetHashrateAvgGHSCustom(time.Duration(0))
			if !ok {
				panic("custom hashrate not found")
			}
			actualHashrate += hr
		}
		return true
	})

	hrError := lib.RelativeError(targetHashrateGHS, actualHashrate)
	hrMsg := fmt.Sprintf("worker %s, target HR %d, actual HR %d, error %.0f%%, threshold(%.0f%%)", dest.Username(), targetHashrateGHS, actualHashrate, hrError*100, hashrateDiffThreshold*100)

	if hrError > hashrateDiffThreshold {
		if actualHashrate < targetHashrateGHS {
			s.log.Warnf("contract is underdelivering: %s", hrMsg)
			return false
		}
		// contract overdelivery is fine for buyer
		s.log.Infof("contract is overdelivering: %s", hrMsg)
	} else {
		s.log.Infof("contract is delivering accurately: %s", hrMsg)
	}

	return true
}

// adjustAllocCollection adjusts percentage for each allocation item so it wont fall in red zone
func (s *GlobalSchedulerV2) adjustAllocCollection(coll *data.AllocCollection, snap *data.AllocSnap) *data.AllocCollection {
	// check if miner amount can be reduced, by allocating more percentage for existing miners
	// if difference is 1 it is likely a deliberate split to avoid red zones
	coll = s.tryReduceMiners(coll)

	// check red zones
	s.tryAdjustRedZones(coll, snap)

	// TODO
	// 0. check is alloc collection violates constraints

	// 1. check if split items could be merged together with constraints

	// 2. apply constraints by adding one more miner or reducing existing miners

	// 3. attempt to drop everything and allocate from scratch and compare (but try to avoid unnecessary reallocations)
	return coll
}

func (s *GlobalSchedulerV2) tryReduceMiners(coll *data.AllocCollection) *data.AllocCollection {
	allocatedHR := 0
	totalHR := coll.GetAllocatedGHS()
	reducedColl := data.NewAllocCollection()

	for _, item := range coll.SortByAllocatedGHSInv() {
		remainingHR := totalHR - allocatedHR
		if lib.AlmostEqual(totalHR, allocatedHR, 0.001) {
			reducedColl.Add(item.MinerID, &data.AllocItem{
				MinerID:    item.MinerID,
				ContractID: item.ContractID,
				Fraction:   0,
				TotalGHS:   item.TotalGHS,
			})
		} else if remainingHR <= item.TotalGHS {
			reducedColl.Add(item.MinerID, &data.AllocItem{
				MinerID:    item.MinerID,
				ContractID: item.ContractID,
				Fraction:   float64(remainingHR) / float64(item.TotalGHS),
				TotalGHS:   item.TotalGHS,
			})
			allocatedHR = totalHR
		} else {
			allocatedHR += item.TotalGHS
			reducedColl.Add(item.MinerID, &data.AllocItem{
				MinerID:    item.MinerID,
				ContractID: item.ContractID,
				Fraction:   1,
				TotalGHS:   item.TotalGHS,
			})
		}
	}

	// only apply if we removed at least one miner from allocation
	// otherwise avoid changing allocation
	if reducedColl.GetZeroAllocatedCount() > 0 {
		s.log.Debugf("redistributed successfully: \n===before\n %s \n===after %s", coll.String(), reducedColl.String())
		coll = reducedColl
	}

	return coll
}

func (s *GlobalSchedulerV2) tryAdjustRedZones(coll *data.AllocCollection, snap *data.AllocSnap) {
	s.log.Debugf("before red zone adjustment: %s", coll.String())

	for _, item := range coll.SortByAllocatedGHS() {
		if lib.AlmostEqual(item.Fraction, 0, 0.01) {
			item.Fraction = 0
			continue
		}
		if lib.AlmostEqual(item.Fraction, 1, 0.01) {
			item.Fraction = 1
			continue
		}

		ok := true

		if item.Fraction < s.poolMinFraction {
			ok = s.adjustLeftRedZone(item, coll)
		} else if item.Fraction > s.poolMaxFraction {
			ok = s.adjustRightRedZone(item, snap, coll)
		}
		if !ok {
			s.log.Warnf("couldn't adjust red zone for minerID(%s), contractID(%s), fraction(%.2f)", item.MinerID, item.ContractID, item.Fraction)
		}
	}
	s.log.Debugf("after red zone adjustment: %s", coll.String())
}

func (s *GlobalSchedulerV2) adjustLeftRedZone(item *data.AllocItem, coll *data.AllocCollection) bool {
	for _, item2 := range coll.GetItems() {
		if item2.MinerID == item.MinerID {
			continue
		}

		f1, f2, ok := FindMidpointSplitWRedzones(s.poolMinFraction, s.poolMaxFraction, item.TotalGHS, item2.TotalGHS, item.AllocatedGHS()+item2.AllocatedGHS())
		if !ok {
			continue
		}

		item.Fraction, item2.Fraction = f1, f2
		return true
	}
	return false
}

func (s *GlobalSchedulerV2) adjustRightRedZone(item *data.AllocItem, snap *data.AllocSnap, coll *data.AllocCollection) bool {
	var existingAndFreeMiners []*data.AllocItem
	_, freeMiners := snap.GetUnallocatedGHS()

	existingAndFreeMiners = append(existingAndFreeMiners, coll.SortByAllocatedGHS()...)
	existingAndFreeMiners = append(existingAndFreeMiners, freeMiners.FilterFullyAvailable().Iter()...)

	for _, item2 := range existingAndFreeMiners {
		if item2.MinerID == item.MinerID {
			continue
		}

		// for existing miners fraction 1 means it is busy
		// for new miners fraction 1 means it is fully available
		// TODO: fix snap.GetUnallocatedGHS()
		if item2.ContractID == "" {
			item2.Fraction = 0
		}

		f1, f2, ok := FindMidpointSplitWRedzones(s.poolMinFraction, s.poolMaxFraction, item.TotalGHS, item2.TotalGHS, item.AllocatedGHS()+item2.AllocatedGHS())
		if !ok {
			continue
		}

		newItem2 := &data.AllocItem{
			MinerID:    item2.MinerID,
			ContractID: item.ContractID,
			Fraction:   item2.Fraction,
			TotalGHS:   item2.TotalGHS,
		}
		coll.Add(item2.MinerID, newItem2)

		item.Fraction, newItem2.Fraction = f1, f2
		return true
	}
	return false
}

// FindMidpointSplitWRedzones solves the system of inequalities:
//
//	totalHR1 * fraction1 + totalHR2 * fraction2 = targetHR
//	minFraction < fraction1 < maxFraction
//	minFraction < fraction2 < maxFraction
//
// returning the midpoint of intervals of fraction1 and fraction2
//
// NB: it does not consider option for allocating miner for 100%
// or 0%. Those cases should be ruled out before using this function
func FindMidpointSplitWRedzones(minFraction, maxFraction float64, totalHR1, totalHR2, targerHR int) (fraction1 float64, fraction2 float64, ok bool) {
	leftEndpointF1 := math.Max((float64(targerHR)-maxFraction*float64(totalHR2))/float64(totalHR1), minFraction)
	rightEndpointF1 := math.Min((float64(targerHR)-minFraction*float64(totalHR2))/float64(totalHR1), maxFraction)
	if leftEndpointF1 > rightEndpointF1 {
		return 0, 0, false
	}

	fraction1 = (leftEndpointF1 + rightEndpointF1) / 2
	fraction2 = (float64(targerHR) - float64(totalHR1)*fraction1) / float64(totalHR2)

	return fraction1, fraction2, true
}
