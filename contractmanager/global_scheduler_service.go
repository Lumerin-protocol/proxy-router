package contractmanager

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/miner"
)

const (
	MIN_DEST_TIME = 2 * time.Minute // minimum time the miner can be pointed to the destination
	MAX_DEST_TIME = 5 * time.Minute // maximum time the miner can be pointed to the destination
)

var (
	ErrNotEnoughHashrate     = errors.New("not enough hashrate")                // simply not enough hashrate
	ErrCannotFindCombination = errors.New("cannot find allocation combination") // hashrate is enough but with given constraint cannot find a working combination of miner alloc items. Adding more miners into system should help
)

type GlobalSchedulerService struct {
	minerCollection interfaces.ICollection[miner.MinerScheduler]
	poolMinDuration time.Duration
	poolMaxDuration time.Duration
	poolMinFraction float64
	poolMaxFraction float64
	log             interfaces.ILogger
}

func NewGlobalScheduler(minerCollection interfaces.ICollection[miner.MinerScheduler], log interfaces.ILogger, poolMinDuration, poolMaxDuration time.Duration) *GlobalSchedulerService {
	instance := &GlobalSchedulerService{
		minerCollection: minerCollection,
		log:             log,
	}
	instance.setPoolDurationConstraints(poolMinDuration, poolMaxDuration)
	return instance
}

func (s *GlobalSchedulerService) setPoolDurationConstraints(min, max time.Duration) {
	s.poolMinDuration, s.poolMaxDuration = min, max
	s.poolMinFraction = float64(min) / float64(min+max)
	s.poolMaxFraction = float64(max) / float64(min+max)
}

func (s *GlobalSchedulerService) Allocate(contractID string, hashrateGHS int, dest interfaces.IDestination) (*AllocCollection, error) {
	snap := s.GetMinerSnapshot()

	remainingHashrate, minerHashrates := snap.GetUnallocatedGHS()
	if remainingHashrate < hashrateGHS {
		return nil, lib.WrapError(ErrNotEnoughHashrate, fmt.Errorf("required %d available %d", hashrateGHS, remainingHashrate))
	}

	combination, isAccurate := s.getAllocateComb(minerHashrates, hashrateGHS)
	if !isAccurate {
		// repeat on available miners only
		// TODO: consider replacing only one alloc item to a miner
		combination, isAccurate = s.getAllocateComb(minerHashrates.FilterFullyAvailable(), hashrateGHS)
		if !isAccurate {
			s.log.Warnf("cannot find accurate combination")
			// return nil, ErrCannotFindCombination
		}
	}

	for _, item := range combination.GetItems() {
		miner, ok := s.minerCollection.Load(item.GetSourceID())
		if !ok {
			s.log.Warnf("unknown miner: %v, skipping", item.GetSourceID())
			continue
		}

		destSplit, err := miner.GetDestSplit().Allocate(contractID, item.Fraction, dest)
		if err != nil {
			s.log.Warnf("failed to allocate miner: %v, skipping...; %w", item.GetSourceID(), err)
			continue
		}

		miner.SetDestSplit(destSplit)
	}

	// pass returnErr whether nil or not;  this way we can attach errors without crashing
	return combination, nil
}

func (s *GlobalSchedulerService) getAllocateComb(minerHashrates *AllocCollection, hashrateGHS int) (col *AllocCollection, isAccurate bool) {
	isAccurate = true

	combination, delta := FindCombinations(minerHashrates, hashrateGHS)

	if delta > 0 {
		// now we need to reduce allocation for the amount of delta
		// there would be two kinds of alloc items
		// 1. the alloc item of the miner that was already allocated to 1 contract
		// 2. the 100% alloc item of the miner that wasn't allocated to contract yet
		//
		// we can only reduce allocation for second kind of miner
		var bestMinerID string
		bestMinerID, ok := s.getBestMinerToReduceHashrate(combination, delta)

		if !ok {
			s.log.Warnf("couldn't find accurate combination")
			isAccurate = false
			bestMinerID, delta = s.getBestMinerToReduceHashrateNotAccurate(combination, delta)
		}

		combination.ReduceMinerAllocation(bestMinerID, delta)
	}

	s.log.Debugf("target to reduce: %d, actual reduced: %d, combination:\n %s", hashrateGHS, delta, combination.String())

	return combination, isAccurate
}

func (s *GlobalSchedulerService) getBestMinerToReduceHashrate(combination *AllocCollection, hrToReduceGHS int) (minerID string, ok bool) {
	var optimalFraction float64 = 0.5

	var bestMinerID string
	var bestMinerFractionDelta float64 = 1

	for _, item := range combination.GetItems() {
		if item.Fraction == 1 {
			fraction := float64(hrToReduceGHS) / float64(item.TotalGHS)
			fractionDelta := math.Abs(fraction - optimalFraction)
			if fraction > s.poolMinFraction &&
				fraction < s.poolMaxFraction &&
				fractionDelta < bestMinerFractionDelta {
				bestMinerID = item.GetSourceID()
				bestMinerFractionDelta = fractionDelta
			}
		}
	}

	return bestMinerID, bestMinerID != ""
}

// getBestMinerToReduceHashrateNotAccurate finds best miner fraction of which could be reduced but not exactly by targetHRToReduceGHS
func (s *GlobalSchedulerService) getBestMinerToReduceHashrateNotAccurate(combination *AllocCollection, targetHRToReduceGHS int) (minerID string, bestHRToReduceGHS int) {
	var (
		bestMinerID       string
		bestHrToReduceGHS int // reduced hashrate of particular miner
		bestDeltaGHS      int // delta between reduced HR of particular miner and targetHRToReduceGHS
	)

	for _, item := range combination.GetItems() {
		if item.Fraction == 1 {
			targetHrGHS := item.TotalGHS - targetHRToReduceGHS
			fraction := float64(targetHrGHS) / float64(item.TotalGHS)
			if fraction < s.poolMinFraction {
				fraction = s.poolMinFraction
			}
			if fraction == 1 {
				fraction = 1
			}
			if fraction > s.poolMaxFraction {
				fraction = s.poolMaxFraction
			}
			actualHrGHS := int(fraction * float64(item.TotalGHS))
			currentHrToReduceGHS := item.TotalGHS - actualHrGHS
			currentDelta := int(math.Abs(float64(currentHrToReduceGHS - targetHRToReduceGHS)))

			if currentDelta < bestDeltaGHS {
				bestDeltaGHS = currentDelta
				bestHrToReduceGHS = currentHrToReduceGHS
				bestMinerID = item.MinerID
			}
		}
	}

	return bestMinerID, bestHrToReduceGHS
}

func (s *GlobalSchedulerService) GetMinerSnapshot() AllocSnap {
	return CreateMinerSnapshot(s.minerCollection)
}

func (s *GlobalSchedulerService) GetUnallocatedHashrateGHS() (int, HashrateList) {
	var unallocatedHashrate int = 0
	var minerHashrates HashrateList

	s.minerCollection.Range(func(miner miner.MinerScheduler) bool {
		hashrate := miner.GetUnallocatedHashrateGHS()
		if hashrate > 0 {
			unallocatedHashrate += hashrate
			// passing to struct to avoid potential race conditions due to hashrate not being constant
			minerHashrates = append(minerHashrates, HashrateListItem{
				Hashrate:      miner.GetUnallocatedHashrateGHS(),
				MinerID:       miner.GetID(),
				TotalHashrate: miner.GetHashRateGHS(),
			})
		}
		return true
	})

	return unallocatedHashrate, minerHashrates
}

func (s *GlobalSchedulerService) UpdateCombination(ctx context.Context, targetHashrateGHS int, dest interfaces.IDestination, contractID string, hashrateDiffThreshold float64) error {
	snapshot := s.GetMinerSnapshot()
	s.log.Info(snapshot.String())

	actualHashrate := 0
	miners, ok := snapshot.Contract(contractID)
	if ok {
		for _, m := range miners.GetItems() {
			actualHashrate += m.AllocatedGHS()
		}
	} else {
		s.log.Warnf("no miner is serving the contract %s", contractID)
	}

	deltaGHS := targetHashrateGHS - actualHashrate
	s.log.Debugf("target hashrate %d, actual hashrate %d, delta %d", targetHashrateGHS, actualHashrate, deltaGHS)
	// check if hashrate increase is available in the system

	if math.Abs(float64(deltaGHS))/float64(targetHashrateGHS) < hashrateDiffThreshold {
		s.log.Debugf("no need to adjust allocation")
		return nil
	}

	if deltaGHS > 0 {
		s.log.Debugf("increasing allocation")
		return s.incAllocation(ctx, snapshot, deltaGHS, dest, contractID)
	} else {
		s.log.Debugf("decreasing allocation")
		return s.decrAllocation(ctx, snapshot, -deltaGHS, contractID)
	}
}

func (s *GlobalSchedulerService) DeallocateContract(ctx context.Context, contractID string) {
	s.log.Infof("deallocating contract %s", contractID)

	snapshot := s.GetMinerSnapshot()
	s.log.Info(snapshot.String())

	minersSnap, ok := snapshot.Contract(contractID)
	if !ok {
		s.log.Warnf("contract (%s) not found", contractID)
		return
	}

	for minerID := range minersSnap.GetItems() {
		miner, ok := s.minerCollection.Load(minerID)
		if !ok {
			s.log.Warnf("miner (%s) is not found", minerID)
			continue
		}

		destSplit, ok := miner.GetDestSplit().RemoveByID(contractID)
		if !ok {
			s.log.Warnf("split (%s) not found", contractID)
			continue
		}

		miner.SetDestSplit(destSplit)
	}
}

// incAllocation increases allocation hashrate prioritizing allocation of existing miners
func (s *GlobalSchedulerService) incAllocation(ctx context.Context, snapshot AllocSnap, addGHS int, dest interfaces.IDestination, contractID string) error {
	remainingToAddGHS := addGHS
	minerIDs := []string{}

	minersSnap, ok := snapshot.Contract(contractID)
	if !ok {
		s.log.Warnf("contract (%s) not found", contractID)
	} else {

		// try to increase allocation in the miners that already serve the contract
		for minerID, minerSnap := range minersSnap.GetItems() {
			miner, ok := s.minerCollection.Load(minerID)
			if !ok {
				s.log.Warnf("miner (%s) is not found", minerID)
				continue
			}

			minerIDs = append(minerIDs, minerID)
			if remainingToAddGHS <= 0 {
				continue
			}

			minerAlloc, ok := snapshot.Miner(minerID)
			if !ok {
				s.log.Warnf("miner (%s) not found")
				continue
			}
			_, unallocatedItem := minerAlloc.GetUnallocatedGHS()
			unallocatedItem.TotalGHS = snapshot.minerIDHashrateGHS[minerID]
			toAllocateGHS := lib.MinInt(remainingToAddGHS, unallocatedItem.AllocatedGHS())
			if toAllocateGHS == 0 {
				continue
			}

			fractionToAdd := float64(toAllocateGHS) / float64(minerSnap.TotalGHS)
			allocationItem, _ := minerAlloc.Get(contractID)
			newFraction := allocationItem.Fraction + fractionToAdd

			if newFraction < s.poolMinFraction {
				continue
			}

			if newFraction > s.poolMaxFraction && newFraction < 1 {
				newFraction = 0.5
				fractionToAdd = newFraction - (1 - unallocatedItem.Fraction)
			}

			split, ok := miner.GetDestSplit().SetFractionByID(contractID, newFraction)
			if !ok {
				s.log.Warnf("split item for contract (%s) not found in miner (%s)", contractID, minerID)
				continue
			}

			miner.SetDestSplit(split)
			remainingToAddGHS -= int(fractionToAdd * float64(minerSnap.TotalGHS))
		}
	}

	if remainingToAddGHS == 0 {
		return nil
	}

	_, err := s.Allocate(contractID, remainingToAddGHS, dest)
	if err != nil {
		return err
	}

	return nil
}

func (s *GlobalSchedulerService) decrAllocation(ctx context.Context, snapshot AllocSnap, removeGHS int, contractID string) error {
	allocSnap, ok := snapshot.Contract(contractID)
	if !ok {
		s.log.Errorf("contract (%s) not found in snap", contractID)
		return nil
	}

	remainingGHS := removeGHS
	for _, item := range allocSnap.SortByAllocatedGHS() {
		if remainingGHS <= 0 {
			break
		}

		miner, ok := s.minerCollection.Load(item.GetSourceID())
		if !ok {
			s.log.Warnf("miner (%s) not found", item.GetSourceID())
			continue
		}

		split := miner.GetDestSplit()
		deallocatedGHS := 0

		if remainingGHS >= item.AllocatedGHS() {
			// remove miner totally
			split, ok = split.RemoveByID(contractID)
			if !ok {
				s.log.Debug("%s", split.String())
				s.log.Warnf("split (%s) not found", contractID)
			}

			deallocatedGHS = item.AllocatedGHS()
		} else {
			newFraction := item.Fraction - float64(remainingGHS)/float64(item.TotalGHS)
			deallocatedGHS = remainingGHS

			if newFraction < s.poolMinFraction {
				split, ok = split.RemoveByID(contractID)
				if !ok {
					s.log.Debugf("Splits:\n%s", split.String())
					s.log.Warnf("split (%s) not found", contractID)
				}
				deallocatedGHS = item.AllocatedGHS()
			} else {
				if newFraction > s.poolMaxFraction {
					newFraction = 0.5
					deallocatedGHS = int(float64(item.TotalGHS) * newFraction)
				}

				split, ok = split.SetFractionByID(contractID, newFraction)
				if !ok {
					s.log.Warnf("split (%s) not found", contractID)
				}
			}
		}

		miner.SetDestSplit(split)
		remainingGHS -= deallocatedGHS
	}

	return nil
}
