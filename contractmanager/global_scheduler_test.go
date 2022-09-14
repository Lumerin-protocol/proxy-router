package contractmanager

import (
	"context"
	"testing"

	"gitlab.com/TitanInd/hashrouter/data"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/miner"
	"gitlab.com/TitanInd/hashrouter/protocol"
)

func CreateMockMinerCollection(contractID string, dest lib.Dest) *data.Collection[miner.MinerScheduler] {
	// destSplit1 := miner.NewDestSplit()
	// destSplit1.Allocate(contractID, 0.5, dest)

	// destSplit2 := miner.NewDestSplit()
	// destSplit2.Allocate(contractID, 0.3, dest)
	DefaultDest, _ := lib.ParseDest("//miner:pwd@default.dest.com:3333")

	miner1 := &protocol.MinerModelMock{
		ID:          "1",
		Dest:        dest,
		HashrateGHS: 10000,
	}
	miner2 := &protocol.MinerModelMock{
		ID:          "2",
		Dest:        dest,
		HashrateGHS: 20000,
	}

	destSplit1 := miner.NewDestSplit()
	destSplit1.Allocate(contractID, 0.5, dest)

	destSplit2 := miner.NewDestSplit()
	destSplit2.Allocate(contractID, 0.3, dest)

	scheduler1 := miner.NewOnDemandMinerScheduler(miner1, destSplit1, &lib.LoggerMock{}, DefaultDest)
	scheduler2 := miner.NewOnDemandMinerScheduler(miner2, destSplit2, &lib.LoggerMock{}, DefaultDest)

	miners := miner.NewMinerCollection()
	miners.Store(scheduler1)
	miners.Store(scheduler2)

	return miners
}

func TestIncAllocation(t *testing.T) {
	addGHS := 5000
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"
	minerIds := []string{"1", "2"}

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{})

	_, err := globalScheduler.incAllocation(context.Background(), minerIds, addGHS, dest, contractID)
	if err != nil {
		t.Fatal(err)
		return
	}

	miner1, _ := miners.Load("1")
	miner2, _ := miners.Load("2")

	destSplit1, _ := miner1.GetDestSplit().GetByID(contractID)
	destSplit2, _ := miner2.GetDestSplit().GetByID(contractID)

	if destSplit1.Percentage != 1 {
		t.Fatal("should use miner which already had been more allocated for the contract")
	}
	if destSplit2.Percentage != 0.3 {
		t.Fatal("should not alter allocation of the second miner")
	}
}

func TestDecrAllocation(t *testing.T) {
	removeGHS := 3000
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"
	minerIds := []string{"1", "2"}

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{})

	_, err := globalScheduler.decrAllocation(context.Background(), minerIds, removeGHS, contractID)
	if err != nil {
		t.Fatal(err)
		return
	}

	miner1, _ := miners.Load("1")
	miner2, _ := miners.Load("2")

	destSplit1, _ := miner1.GetDestSplit().GetByID(contractID)
	destSplit2, _ := miner2.GetDestSplit().GetByID(contractID)

	if destSplit1.Percentage != 0.2 {
		t.Fatal("should use miner which was the least allocated for the contract")
	}
	if destSplit2.Percentage != 0.3 {
		t.Fatal("should not alter allocation of the second miner")
	}
}
