package contractmanager

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

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

	scheduler1 := miner.NewOnDemandMinerScheduler(miner1, destSplit1, &lib.LoggerMock{}, DefaultDest, 0, 0, 0)
	scheduler2 := miner.NewOnDemandMinerScheduler(miner2, destSplit2, &lib.LoggerMock{}, DefaultDest, 0, 0, 0)
	scheduler3 := miner.NewOnDemandMinerScheduler(miner3, destSplit3, &lib.LoggerMock{}, DefaultDest, 0, 0, 0)

	miners := miner.NewMinerCollection()
	miners.Store(scheduler1)
	miners.Store(scheduler2)
	miners.Store(scheduler3)

	return miners
}

// TestAllocation50percent ensures that the allocation prefers to do a split around 50% of miner's hashpower
// because it allows for longer cycle time
func TestAllocation50percent(t *testing.T) {
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"
	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{}, 0, 0)

	contract2ID := "test-contract-2"
	hrGHS := 5000

	col, err := globalScheduler.Allocate(contract2ID, hrGHS, dest)
	if err != nil {
		t.Error(err)
	}
	t.Log(col.String())

	allocItem, ok := col.Get("1")
	if !ok || allocItem.Fraction != 0.5 {
		t.Errorf("miner 1 should be allocated for 0.5 of hashpower")
	}
}

func TestAllocationPreferSingleMiner(t *testing.T) {
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"
	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{}, 2*time.Minute, 5*time.Minute)

	contract2ID := "test-contract-2"
	hrGHS := 10000

	col, err := globalScheduler.Allocate(contract2ID, hrGHS, dest)
	if err != nil {
		t.Error(err)
	}

	miner, ok := col.Get("3")
	if !ok {
		t.Errorf("should use a fully vacant miner")
	}
	expFraction := float64(hrGHS) / float64(miner.TotalGHS)
	if math.Abs((miner.Fraction-expFraction)/miner.Fraction) > 0.001 {
		t.Errorf("incorrect fraction: expected (%.2f) actual (%.2f)\n %s", expFraction, miner.Fraction, miner)
	}
}

func TestAllocationReduce(t *testing.T) {
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"
	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{}, 0, 0)

	contract2ID := "test-contract-2"
	hrGHS := 5000

	col, err := globalScheduler.Allocate(contract2ID, hrGHS, dest)
	if err != nil {
		t.Error(err)
	}
	t.Log(col.String())

	allocItem, ok := col.Get("1")
	if !ok || allocItem.Fraction != 0.5 {
		t.Errorf("miner 1 should be allocated for 0.5 of hashpower")
	}
}

func TestIncAllocation(t *testing.T) {
	addGHS := 5000
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{}, 0, 0)
	snapshot := globalScheduler.GetMinerSnapshot()

	_, err := globalScheduler.incAllocation(context.Background(), snapshot, addGHS, dest, contractID)
	if err != nil {
		t.Fatal(err)
	}

	snapshot2 := globalScheduler.GetMinerSnapshot()
	list, ok := snapshot2.Contract(contractID)
	if !ok {
		t.Fatalf("contract should show up in the snapshot")
	}

	totalGHS := 0
	for _, item := range list.GetItems() {
		totalGHS += item.AllocatedGHS()
	}

	expectedGHS := 16000

	if totalGHS != expectedGHS {
		t.Fatalf("total hashrate (%v) should be %v", totalGHS, expectedGHS)
	}
}

func TestIncAllocationAddMiner(t *testing.T) {
	addGHS := 20000
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{}, 0, 0)
	snapshot := globalScheduler.GetMinerSnapshot()

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
	t.Skip()
	removeGHS := 3000
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{}, 0, 0)
	snapshot := globalScheduler.GetMinerSnapshot()

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
	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{}, 0, 0)
	snapshot := globalScheduler.GetMinerSnapshot()

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

func TestGetMinerSnapshot(t *testing.T) {
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")

	miner1 := &protocol.MinerModelMock{
		ID:          "1",
		Dest:        dest,
		HashrateGHS: 10000,
		ConnectedAt: time.Now().Add(-time.Hour),
	}
	miner2 := &protocol.MinerModelMock{
		ID:          "2",
		Dest:        dest,
		HashrateGHS: 20000,
		ConnectedAt: time.Now(),
	}

	vettingPeriod := time.Second * 10

	scheduler1 := miner.NewOnDemandMinerScheduler(miner1, miner.NewDestSplit(), &lib.LoggerMock{}, dest, vettingPeriod, 0, 0)
	scheduler2 := miner.NewOnDemandMinerScheduler(miner2, miner.NewDestSplit(), &lib.LoggerMock{}, dest, vettingPeriod, 0, 0)

	miners := miner.NewMinerCollection()
	miners.Store(scheduler1)
	miners.Store(scheduler2)

	globalScheduler := NewGlobalScheduler(miners, &lib.LoggerMock{}, 0, 0)
	snapshot := globalScheduler.GetMinerSnapshot()

	if len(snapshot.minerIDHashrateGHS) != 1 {
		t.Fatal("should filter out recently connected miner")
	}
	if _, ok := snapshot.minerIDHashrateGHS["1"]; !ok {
		t.Fatal("a miner 1 should be available")
	}
}
