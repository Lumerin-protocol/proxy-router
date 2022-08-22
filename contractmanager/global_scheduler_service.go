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

func (s *GlobalSchedulerService) Allocate(hashrate uint64, dest interfaces.IDestination) error {
	remainingHashrate, minerHashrates := s.GetUnallocatedHashrate()
	if remainingHashrate < hashrate {
		return ErrNotEnoughHashrate
	}

	combination := FindCombinations(minerHashrates, hashrate)
	for _, item := range combination {
		miner, ok := s.minerCollection.Load(item.GetSourceID())
		if !ok {
			panic("miner not found")
		}
		err := miner.Allocate(item.GetPercentage(), dest)
		if err != nil {
			panic(err)
		}
	}

	return nil
}

func (s *GlobalSchedulerService) GetUnallocatedHashrate() (uint64, HashrateList) {
	var unallocatedHashrate uint64 = 0
	var minerHashrates HashrateList

	s.minerCollection.Range(func(miner miner.MinerScheduler) bool {
		unallocatedHashrate = miner.GetUnallocatedHashrate()
		if unallocatedHashrate > 0 {
			unallocatedHashrate += unallocatedHashrate
			// passing to struct to avoid potential race conditions due to hashrate not being constant
			minerHashrates = append(minerHashrates, HashrateListItem{
				Hashrate:      unallocatedHashrate,
				MinerID:       miner.GetID(),
				TotalHashrate: miner.GetHashRate(),
			})
		}
		return true
	})

	return unallocatedHashrate, minerHashrates
}
