package contractmanager

import (
	"context"
	"errors"
	"fmt"
	"math"

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

	s.log.Debug(combination.String())

	for _, item := range combination {
		miner, ok := s.minerCollection.Load(item.GetSourceID())
		if !ok {
			//just logging error message because the miner might disconnect
			s.log.Warnf("unknown miner: %v, skipping", item.GetSourceID())
			continue
		}
		_, err := miner.Allocate(contractID, item.GetPercentage(), dest)
		if err != nil {
			s.log.Warnf("failed to allocate miner: %v, skipping...; %w", item.GetSourceID(), err)
			continue
		}
	}

	//pass returnErr whether nil or not;  this way we can attach errors without crashing
	return combination, nil
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
	miners, ok := snapshot.Contract(contractID)
	if !ok {
		s.log.Warnf("contract not found %s", contractID)
		return minerIDs, nil
	}

	actualHashrate := 0
	for _, m := range miners {
		actualHashrate += m.AllocatedGHS()
	}

	deltaGHS := targetHashrateGHS - actualHashrate
	s.log.Debugf("target hashrate %d, actual hashrate %d, delta %d", targetHashrateGHS, actualHashrate, deltaGHS)
	// check if hashrate increase is available in the system

	if math.Abs(float64(deltaGHS))/float64(targetHashrateGHS) < HASHRATE_DIFF_THRESHOLD {
		s.log.Debugf("no need to update hashrate")
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
func (s *GlobalSchedulerService) incAllocation(ctx context.Context, snapshot AllocSnap, addGHS int, dest lib.Dest, contractID string) ([]string, error) {
	remainingToAddGHS := addGHS

	minersSnap, ok := snapshot.Contract(contractID)
	if !ok {
		s.log.DPanicf("contract (%s) not found", contractID)
	}

	minerIDs := []string{}

	// try to increase allocation in the miners that already serve the contract
	for minerID, minerSnap := range minersSnap {
		miner, ok := s.minerCollection.Load(minerID)
		if !ok {
			s.log.Warnf("miner (%s) is not found", minerID)
			continue
		}

		minerIDs = append(minerIDs, minerID)
		if remainingToAddGHS <= 0 {
			continue
		}

		availableGHS := minerSnap.AvailableGHS()
		toAllocateGHS := lib.MinInt(remainingToAddGHS, availableGHS)
		if toAllocateGHS == 0 {
			continue
		}

		fractionToAdd := float64(toAllocateGHS) / float64(minerSnap.TotalGHS)

		miner.GetDestSplit().IncreaseAllocation(contractID, fractionToAdd)
		remainingToAddGHS -= toAllocateGHS

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

func (s *GlobalSchedulerService) decrAllocation(ctx context.Context, snapshot AllocSnap, removeGHS int, contractID string) ([]string, error) {
	allocSnap, ok := snapshot.Contract(contractID)
	if !ok {
		s.log.DPanicf("contract (%s) not found in snap", contractID)
		return nil, nil
	}

	minerIDs := []string{}
	remainingGHS := removeGHS
	for _, item := range allocSnap.SortByAllocatedGHS() {
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

		if remainingGHS >= item.AllocatedGHS() {
			// remove miner totally
			ok := split.RemoveByID(contractID)
			if !ok {
				s.log.Warnf("split (%s) not found", contractID)
			}
			removeMinerGHS = item.AllocatedGHS()

		} else {
			removeMinerFraction := float64(remainingGHS) / float64(item.TotalGHS)
			// if removeMinerFraction < miner.Min
			newFraction := item.Fraction - removeMinerFraction
			split.SetFractionByID(contractID, newFraction)
			removeMinerGHS = remainingGHS
			minerIDs = append(minerIDs, item.MinerID)
		}

		remainingGHS -= removeMinerGHS
	}

	if remainingGHS != 0 {
		err := fmt.Errorf("deallocation fault, remainingGHS %d, allocSnap %+v", remainingGHS, allocSnap)
		s.log.DPanic(err)
		return nil, err
	}

	return minerIDs, nil
}
