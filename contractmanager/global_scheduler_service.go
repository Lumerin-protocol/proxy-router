package contractmanager

import (
	"errors"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/miner"
)

var (
	ErrNotEnoughHashrate = errors.New("not enough hashrate")
)

type GlobalSchedulerService struct {
	minerCollection *miner.MinerRepo
}

func (s *GlobalSchedulerService) Allocate(hashrateGHS int, dest interfaces.IDestination) (HashrateList, error) {
	remainingHashrate, minerHashrates := s.GetUnallocatedHashrateGHS()
	if remainingHashrate < hashrateGHS {
		return nil, ErrNotEnoughHashrate
	}

	combination := FindCombinations(minerHashrates, hashrateGHS)
	for _, item := range combination {
		miner, ok := s.minerCollection.Load(item.GetSourceID())
		if !ok {
			panic("miner not found")
		}
		splitPtr, err := miner.Allocate(item.GetPercentage(), dest)
		if err != nil {
			panic(err)
		}
		item.SplitPtr = splitPtr
	}

	return combination, nil
}

func (s *GlobalSchedulerService) GetUnallocatedHashrateGHS() (int, HashrateList) {
	var unallocatedHashrate int = 0
	var minerHashrates HashrateList

	s.minerCollection.Range(func(miner miner.MinerScheduler) bool {
		unallocatedHashrate = miner.GetUnallocatedHashrateGHS()
		if unallocatedHashrate > 0 {
			unallocatedHashrate += unallocatedHashrate
			// passing to struct to avoid potential race conditions due to hashrate not being constant
			minerHashrates = append(minerHashrates, HashrateListItem{
				Hashrate:      unallocatedHashrate,
				MinerID:       miner.GetID(),
				TotalHashrate: miner.GetHashRateGHS(),
			})
		}
		return true
	})

	return unallocatedHashrate, minerHashrates
}
