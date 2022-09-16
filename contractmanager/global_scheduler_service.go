package contractmanager

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/miner"
)

const (
	HASHRATE_DIFF_THRESHOLD = 0.1
)

var (
	ErrNotEnoughHashrate = errors.New("not enough hashrate")
)

type GlobalSchedulerService struct {
	minerCollection interfaces.ICollection[miner.MinerScheduler]
	log             interfaces.ILogger
}

func NewGlobalScheduler(minerCollection interfaces.ICollection[miner.MinerScheduler], log interfaces.ILogger) *GlobalSchedulerService {
	return &GlobalSchedulerService{
		minerCollection: minerCollection,
		log:             log,
	}
}

func (s *GlobalSchedulerService) Allocate(contractID string, hashrateGHS int, dest interfaces.IDestination) (HashrateList, error) {
	remainingHashrate, minerHashrates := s.GetUnallocatedHashrateGHS()
	if remainingHashrate < hashrateGHS {
		return nil, lib.WrapError(ErrNotEnoughHashrate, fmt.Errorf("required %d available %d", hashrateGHS, remainingHashrate))
	}

	combination := FindCombinations(minerHashrates, hashrateGHS)
	for i, item := range combination {
		miner, ok := s.minerCollection.Load(item.GetSourceID())
		if !ok {
			//just logging error message because the miner might disconnect
			s.log.Warnf("unknown miner: %v, skipping", item.GetSourceID())
			continue
		}
		splitPtr, err := miner.Allocate(contractID, item.GetPercentage(), dest)
		if err != nil {
			s.log.Warnf("failed to allocate miner: %v, skipping...; %w", item.GetSourceID(), err)
			continue
		}
		combination[i].SplitPtr = splitPtr
	}

	//pass returnErr whether nil or not;  this way we can attach errors without crashing
	return combination, nil
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
	totalHashrate := 0

	for _, minerID := range minerIDs {
		miner, ok := s.minerCollection.Load(minerID)
		if !ok {
			continue
		}
		destSplit, ok := miner.GetDestSplit().GetByID(contractID)
		if !ok {
			s.log.Warn("cannot find split", contractID)
		}
		actualHashrateGHS := int(float64(miner.GetHashRateGHS()) * destSplit.Percentage)
		totalHashrate += actualHashrateGHS
	}

	deltaGHS := targetHashrateGHS - totalHashrate
	s.log.Debug("target hashrate %d, actual hashrate %d, delta %d", targetHashrateGHS, totalHashrate, deltaGHS)
	// check if hashrate increase is available in the system

	if math.Abs(float64(deltaGHS))/float64(targetHashrateGHS) < HASHRATE_DIFF_THRESHOLD {
		return minerIDs, nil
	}

	if deltaGHS > 0 {
		return s.incAllocation(ctx, minerIDs, deltaGHS, dest, contractID)
	} else {
		return s.decrAllocation(ctx, minerIDs, -deltaGHS, contractID)
	}
}

func (s *GlobalSchedulerService) DeallocateContract(minerIDs []string, contractID string) {
	for _, minerID := range minerIDs {
		miner, ok := s.minerCollection.Load(minerID)
		if !ok {
			s.log.Warnf("allocation error: miner (%s) not found (%s)", minerID, contractID)
			continue
		}

		ok = miner.Deallocate(contractID)
		if !ok {
			s.log.Warnf("allocation error: miner (%s) is not fulfilling this contract (%s)", minerID, contractID)
		}
	}
}

// incAllocation increases allocation hashrate prioritizing allocation of existing miners
func (s *GlobalSchedulerService) incAllocation(ctx context.Context, minerIDs []string, addGHS int, dest lib.Dest, contractID string) ([]string, error) {
	remainingToAddGHS := addGHS

	// try to increase allocation in the miners that already serve the contract
	for _, minerID := range minerIDs {
		miner, ok := s.minerCollection.Load(minerID)
		if !ok {
			continue
		}

		availableFraction := float64(miner.GetUnallocatedHashrateGHS()) / float64(miner.GetHashRateGHS())
		availableHashrateGHS := int(float64(miner.GetHashRateGHS()) * availableFraction)
		hashrateToAllocateGHS := lib.MinInt(remainingToAddGHS, availableHashrateGHS)
		if hashrateToAllocateGHS == 0 {
			continue
		}

		fractionToAdd := float64(hashrateToAllocateGHS) / float64(miner.GetHashRateGHS())
		miner.GetDestSplit().IncreaseAllocation(contractID, fractionToAdd)
		remainingToAddGHS -= hashrateToAllocateGHS
		if remainingToAddGHS == 0 {
			break
		}
	}

	if remainingToAddGHS == 0 {
		return minerIDs, nil
	}

	newHashrateList, err := s.Allocate(contractID, remainingToAddGHS, dest)
	if err != nil {
		return nil, err
	}
	addMinerIDs := make([]string, newHashrateList.Len())
	for i, item := range newHashrateList {
		minerIDs[i] = item.MinerID
	}

	newCombination := append(minerIDs, addMinerIDs...)
	return newCombination, nil
}

func (s *GlobalSchedulerService) decrAllocation(ctx context.Context, oldMinerIDs []string, removeGHS int, contractID string) ([]string, error) {
	newHashrateList := HashrateList{}

	for _, minerID := range oldMinerIDs {
		miner, ok := s.minerCollection.Load(minerID)
		if !ok {
			continue
		}

		allocated, ok := miner.GetDestSplit().GetByID(contractID)
		if !ok {
			s.log.Warnf("miner (%s) that was fulfilling contract (%s) not found", minerID, contractID)
			continue
		}
		availableFraction := 1 - allocated.Percentage
		availableHashrateGHS := int(float64(miner.GetHashRateGHS()) * availableFraction)
		newHashrateList = append(newHashrateList, HashrateListItem{
			MinerID:       minerID,
			Hashrate:      availableHashrateGHS,
			TotalHashrate: miner.GetHashRateGHS(),
			Percentage:    availableFraction,
		})
	}

	sort.Sort(newHashrateList)

	remainingGHS := removeGHS
	for _, item := range newHashrateList {
		if remainingGHS <= 0 {
			break
		}

		miner, ok := s.minerCollection.Load(item.MinerID)
		if !ok {
			s.log.Warnf("miner (%s) not found", item.MinerID)
			continue
		}

		split := miner.GetDestSplit()
		removeMinerGHS := 0

		if remainingGHS >= item.Hashrate {
			// remove miner totally
			ok := split.RemoveByID(contractID)
			if !ok {
				s.log.Warnf("split (%s) not found", contractID)
			}
			removeMinerGHS = item.Hashrate

			for i, ID := range oldMinerIDs {
				if ID == item.MinerID {
					oldMinerIDs = append(oldMinerIDs[:i], oldMinerIDs[i+1:]...)
					break
				}
			}
		} else {
			removeMinerFraction := float64(remainingGHS) / float64(item.TotalHashrate)
			// if removeMinerFraction < miner.Min
			newFraction := item.Percentage - removeMinerFraction
			split.SetFractionByID(contractID, newFraction)
			removeMinerGHS = remainingGHS
		}

		remainingGHS -= removeMinerGHS
	}

	if remainingGHS != 0 {
		err := fmt.Errorf("deallocation fault, remainingGHS %d, hashrateList %+v", remainingGHS, newHashrateList)
		s.log.DPanic(err)
		return nil, err
	}

	return oldMinerIDs, nil
}
