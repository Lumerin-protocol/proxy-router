package contractmanager

import (
	"context"
	"fmt"
	"testing"

	"gitlab.com/TitanInd/hashrouter/data"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/miner"
	"gitlab.com/TitanInd/hashrouter/protocol"
)

func CreateMockMinerCollection(contractID string, dest lib.Dest) *data.Collection[miner.MinerScheduler] {
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
	miner3 := &protocol.MinerModelMock{
		ID:          "3",
		Dest:        dest,
		HashrateGHS: 30000,
	}

	destSplit1 := miner.NewDestSplit()
	_, _ = destSplit1.Allocate(contractID, 0.5, dest)

	destSplit2 := miner.NewDestSplit()
	_, _ = destSplit2.Allocate(contractID, 0.3, dest)

	destSplit3 := miner.NewDestSplit()

	scheduler1 := miner.NewOnDemandMinerScheduler(miner1, destSplit1, &lib.LoggerMock{}, DefaultDest)
	scheduler2 := miner.NewOnDemandMinerScheduler(miner2, destSplit2, &lib.LoggerMock{}, DefaultDest)
	scheduler3 := miner.NewOnDemandMinerScheduler(miner3, destSplit3, &lib.LoggerMock{}, DefaultDest)

	miners := miner.NewMinerCollection()
	miners.Store(scheduler1)
	miners.Store(scheduler2)
	miners.Store(scheduler3)

	return miners
}

func TestIncAllocation(t *testing.T) {
	addGHS := 5000
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{})
	snapshot := CreateMinerSnapshot(miners)

	fmt.Print(snapshot.String())
	_, err := globalScheduler.incAllocation(context.Background(), snapshot, addGHS, dest, contractID)
	if err != nil {
		t.Fatal(err)
		return
	}

	miner1, _ := miners.Load("1")
	miner2, _ := miners.Load("2")

	destSplit1, _ := miner1.GetDestSplit().GetByID(contractID)
	destSplit2, _ := miner2.GetDestSplit().GetByID(contractID)

	if destSplit1.Percentage != 1 {
		t.Fatalf("should use miner which already had been more allocated for the contract %v", destSplit1)
	}
	if destSplit2.Percentage != 0.3 {
		t.Fatalf("should not alter allocation of the second miner %v", destSplit2)
	}
}

func TestIncAllocationAddMiner(t *testing.T) {
	addGHS := 20000
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{})
	snapshot := CreateMinerSnapshot(miners)

	_, err := globalScheduler.incAllocation(context.Background(), snapshot, addGHS, dest, contractID)
	if err != nil {
		t.Fatal(err)
		return
	}

	miner1, _ := miners.Load("1")
	miner2, _ := miners.Load("2")
	miner3, _ := miners.Load("3")

	destSplit1, _ := miner1.GetDestSplit().GetByID(contractID)
	destSplit2, _ := miner2.GetDestSplit().GetByID(contractID)
	destSplit3, _ := miner3.GetDestSplit().GetByID(contractID)

	if destSplit1.Percentage != 1 {
		t.Fatal("should use this contract's most already allocated miner")
	}
	if destSplit2.Percentage != 1 {
		t.Fatal("should use this contract's second most already allocated miner")
	}
	if destSplit3.Percentage == 0.1 {
		t.Fatal("should add new miner")
	}
}

func TestDecrAllocation(t *testing.T) {
	removeGHS := 3000
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{})
	snapshot := CreateMinerSnapshot(miners)

	_, err := globalScheduler.decrAllocation(context.Background(), snapshot, removeGHS, contractID)
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

func TestDecrAllocationRemoveMiner(t *testing.T) {
	removeGHS := 5000
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{})
	snapshot := CreateMinerSnapshot(miners)

	_, err := globalScheduler.decrAllocation(context.Background(), snapshot, removeGHS, contractID)
	if err != nil {
		t.Fatal(err)
		return
	}

	miner1, _ := miners.Load("1")
	miner2, _ := miners.Load("2")

	destSplit1, ok1 := miner1.GetDestSplit().GetByID(contractID)
	destSplit2, ok2 := miner2.GetDestSplit().GetByID(contractID)

	if ok1 {
		fmt.Println(destSplit1)
		t.Fatal("should remove miner which was the least allocated for the contract")
	}
	if !ok2 {
		t.Fatal("should not remove second miner")
	}
	if destSplit2.Percentage != 0.3 {
		t.Fatal("should not alter allocation of the second miner")
	}
}
