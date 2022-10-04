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
	HASHRATE_DIFF_THRESHOLD = 0.1
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
		miner, ok := s.minerCollection.Load(item.GetSourceId())
		if !ok {
			//just logging error message because the miner might disconnect
			s.log.Warnf("unknown miner: %v, skipping", item.GetSourceId())
			continue
		}
		_, err := miner.Allocate(contractID, item.Fraction, dest)
		if err != nil {
			s.log.Warnf("failed to allocate miner: %v, skipping...; %w", item.GetSourceId(), err)
			continue
		}
	}

	// pass returnErr whether nil or not;  this way we can attach errors without crashing
	return combination, nil
}

func (s *GlobalSchedulerService) getAllocateComb(minerHashrates *AllocCollection, hashrateGHS int) (col *AllocCollection, isAccurate bool) {
	combination, delta := FindCombinations(minerHashrates, hashrateGHS)
	s.log.Debug(combination.String())

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
			// TODO: consider replacing the largest alloc item with a new miner
			// items := combination.SortByAllocatedGHS()
			// largestAllocItem := items[len(items)-1]
			// items[largestAllocItem.GetSourceId()] =
			return combination, false
		}

		combination.ReduceMinerAllocation(bestMinerID, delta)
	}

	return combination, true
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
				bestMinerID = item.GetSourceId()
				bestMinerFractionDelta = fractionDelta
			}
		}
	}

	return bestMinerID, bestMinerID != ""
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

func (s *GlobalSchedulerService) UpdateCombination(ctx context.Context, minerIDs []string, targetHashrateGHS int, dest lib.Dest, contractID string) ([]string, error) {
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

	if math.Abs(float64(deltaGHS))/float64(targetHashrateGHS) < HASHRATE_DIFF_THRESHOLD {
		s.log.Debugf("no need to adjust allocation")
		return minerIDs, nil
	}

	if deltaGHS > 0 {
		s.log.Debugf("increasing allocation")
		return s.incAllocation(ctx, snapshot, deltaGHS, dest, contractID)
	} else {
		s.log.Debugf("decreasing allocation")
		return s.decrAllocation(ctx, snapshot, -deltaGHS, contractID)
	}
}

func (s *GlobalSchedulerService) DeallocateContract(minerIDs []string, contractID string) {
	for _, minerID := range minerIDs {
		miner, ok := s.minerCollection.Load(minerID)
		if !ok {
			s.log.Warnf("deallocation error: miner (%s) not found (%s)", minerID, contractID)
			continue
		}

		ok = miner.Deallocate(contractID)
		if !ok {
			s.log.Warnf("deallocation error: miner (%s) is not fulfilling this contract (%s)", minerID, contractID)
		}

	}
}

// incAllocation increases allocation hashrate prioritizing allocation of existing miners
func (s *GlobalSchedulerService) incAllocation(ctx context.Context, snapshot AllocSnap, addGHS int, dest lib.Dest, contractID string) ([]string, error) {
	remainingToAddGHS := addGHS
	minerIDs := []string{}

	minersSnap, ok := snapshot.Contract(contractID)
	if !ok {
		s.log.Errorf("contract (%s) not found", contractID)
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
				s.log.DPanicf("miner (%s) not found")
			}
			_, allocItem := minerAlloc.GetUnallocatedGHS()
			allocItem.TotalGHS = snapshot.minerIDHashrateGHS[minerID]
			toAllocateGHS := lib.MinInt(remainingToAddGHS, allocItem.AllocatedGHS())
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
				fractionToAdd = s.poolMaxFraction - allocationItem.Fraction
			}

			miner.GetDestSplit().IncreaseAllocation(contractID, fractionToAdd)
			remainingToAddGHS -= int(fractionToAdd * float64(minerSnap.TotalGHS))
		}
	}

	if remainingToAddGHS == 0 {
		return minerIDs, nil
	}

	newHashrateList, err := s.Allocate(contractID, remainingToAddGHS, dest)
	if err != nil {
		return nil, err
	}
	addMinerIDs := newHashrateList.IDs()

	newCombination := append(minerIDs, addMinerIDs...)
	return newCombination, nil
}

func (s *GlobalSchedulerService) decrAllocation(ctx context.Context, snapshot AllocSnap, removeGHS int, contractID string) ([]string, error) {
	allocSnap, ok := snapshot.Contract(contractID)
	if !ok {
		s.log.Errorf("contract (%s) not found in snap", contractID)
		return nil, nil
	}

	minerIDs := []string{}
	remainingGHS := removeGHS
	for _, item := range allocSnap.SortByAllocatedGHS() {
		if remainingGHS <= 0 {
			break
		}

		miner, ok := s.minerCollection.Load(item.GetSourceId())
		if !ok {
			s.log.Warnf("miner (%s) not found", item.GetSourceId())
			continue
		}

		split := miner.GetDestSplit()
		removeMinerGHS := 0

		if remainingGHS >= item.AllocatedGHS() {
			// remove miner totally
			ok := split.RemoveByID(contractID)
			if !ok {
				s.log.Warnf("split (%s) not found", contractID)
			}
			removeMinerGHS = item.AllocatedGHS()

		} else {
			removeMinerFraction := float64(remainingGHS) / float64(item.TotalGHS)

			newFraction := item.Fraction - removeMinerFraction
			removeMinerGHS = remainingGHS

			if newFraction < s.poolMinFraction {
				ok := split.RemoveByID(contractID)
				if !ok {
					s.log.Warnf("split (%s) not found", contractID)
				}
				removeMinerGHS = item.AllocatedGHS()
			}

			if newFraction > s.poolMaxFraction {
				newFraction = 0.5
				removeMinerGHS = int(float64(item.TotalGHS) * newFraction)
			}

			split.SetFractionByID(contractID, newFraction)
			minerIDs = append(minerIDs, item.GetSourceId())
		}

		remainingGHS -= removeMinerGHS
	}

	// if remainingGHS != 0 {
	// 	err := fmt.Errorf("deallocation fault, remainingGHS %d, allocSnap %+v", remainingGHS, allocSnap)
	// 	s.log.DPanic(err)
	// 	return nil, err
	// }

	return minerIDs, nil
}
