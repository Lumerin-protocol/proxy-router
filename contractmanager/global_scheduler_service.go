package contractmanager

import (
	"errors"
	"fmt"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/miner"
)

var (
	ErrNotEnoughHashrate = errors.New("not enough hashrate")
)

type GlobalSchedulerService struct {
	minerCollection interfaces.ICollection[miner.MinerScheduler]
}

func NewGlobalScheduler(minerCollection interfaces.ICollection[miner.MinerScheduler]) *GlobalSchedulerService {
	return &GlobalSchedulerService{
		minerCollection,
	}
}

func (s *GlobalSchedulerService) Allocate(hashrateGHS int, dest interfaces.IDestination) (HashrateList, error) {
	remainingHashrate, minerHashrates := s.GetUnallocatedHashrateGHS()
	if remainingHashrate < hashrateGHS {
		return nil, lib.WrapError(ErrNotEnoughHashrate, fmt.Errorf("required %d available %d", hashrateGHS, remainingHashrate))
	}

	combination := FindCombinations(minerHashrates, hashrateGHS)
	for i, item := range combination {
		miner, ok := s.minerCollection.Load(item.GetSourceID())
		if !ok {
			panic("miner not found")
		}
		splitPtr, err := miner.Allocate(item.GetPercentage(), dest)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+#v", splitPtr)
		combination[i].SplitPtr = splitPtr
	}

	fmt.Printf("%+#v", combination)

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
