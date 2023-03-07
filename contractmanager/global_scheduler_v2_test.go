package contractmanager

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	destSplit1, _ := miner.NewDestSplit().Allocate(contractID, 0.5, dest, nil)
	destSplit2, _ := miner.NewDestSplit().Allocate(contractID, 0.3, dest, nil)
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

func TestAllocationPreferSingleMiner(t *testing.T) {
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"
	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalSchedulerV2(miners, &lib.LoggerMock{}, 2*time.Minute, 5*time.Minute, 0, 1)

	contract2ID := "test-contract-2"
	hrGHS := 10000

	err := globalScheduler.update(contract2ID, hrGHS, dest, nil)
	if err != nil {
		t.Error(err)
	}

	col, ok := globalScheduler.GetMinerSnapshot().Contract(contract2ID)
	if !ok {
		t.Fatalf("contract not found")
	}

	miner, ok := col.Get("3")
	if !ok {
		t.Fatalf("should use a fully vacant miner")
	}

	expFraction := float64(hrGHS) / float64(miner.TotalGHS)
	if math.Abs((miner.Fraction-expFraction)/miner.Fraction) > 0.001 {
		t.Errorf("incorrect fraction: expected (%.2f) actual (%.2f)\n %s", expFraction, miner.Fraction, miner)
	}
}

func TestAllocationShouldntSplitBetweenTwoContracts(t *testing.T) {
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"
	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalSchedulerV2(miners, &lib.LoggerMock{}, 0, 0, 0, 1)

	contract2ID := "test-contract-2"
	hrGHS := 5000

	err := globalScheduler.update(contract2ID, hrGHS, dest, nil)
	if err != nil {
		t.Fatal(err)
	}

	col, ok := globalScheduler.GetMinerSnapshot().Contract(contract2ID)
	if !ok {
		t.Fatalf("contract not found")
	}
	t.Log(col.String())

	allocItem, ok := col.Get("3")
	if !ok || allocItem.ContractID != contract2ID {
		t.Errorf("should use miner 3 because it is the only vacant")
	}
}

func TestIncAllocation(t *testing.T) {
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"
	expectedGHS := 16000

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalSchedulerV2(miners, &lib.LoggerMock{}, 0, 0, 0, 1)

	err := globalScheduler.update(contractID, expectedGHS, dest, nil)
	if err != nil {
		t.Fatal(err)
	}

	snapshot2 := globalScheduler.GetMinerSnapshot()
	list, ok := snapshot2.Contract(contractID)
	if !ok {
		t.Fatalf("contract should show up in the snapshot")
	}

	if list.GetAllocatedGHS() != expectedGHS {
		t.Fatalf("total hashrate (%d) should be %d", list.GetAllocatedGHS(), expectedGHS)
	}
}

func TestIncAllocationNotEnoughHR(t *testing.T) {
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"
	contractGHS := 100000

	miners := CreateMockMinerCollection(contractID, dest)
	expectedGHS := 0
	miners.Range(func(item miner.MinerScheduler) bool {
		expectedGHS += item.GetHashRateGHS()
		return true
	})

	globalScheduler := NewGlobalSchedulerV2(miners, &lib.LoggerMock{}, 0, 0, 0, 1)

	err := globalScheduler.update(contractID, contractGHS, dest, nil)
	if err != nil {
		t.Fatal(err)
	}

	snapshot2 := globalScheduler.GetMinerSnapshot()
	list, ok := snapshot2.Contract(contractID)
	if !ok {
		t.Fatalf("contract should show up in the snapshot")
	}

	if list.GetAllocatedGHS() != expectedGHS {
		t.Fatalf("total hashrate (%d) should be %d", list.GetAllocatedGHS(), expectedGHS)
	}
}

func TestIncAllocationAddMiner(t *testing.T) {
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"
	newGHS := 31000
	minPoolDuration, maxPoolDuration := 2*time.Minute, 7*time.Minute

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalSchedulerV2(miners, lib.NewTestLogger(), minPoolDuration, maxPoolDuration, 0.1, 1)

	err := globalScheduler.update(contractID, newGHS, dest, nil)
	if err != nil {
		t.Fatal(err)
	}

	miner1, _ := miners.Load("1")
	miner2, _ := miners.Load("2")
	miner3, _ := miners.Load("3")

	destSplit1, _ := miner1.GetDestSplit().GetByID(contractID)
	destSplit2, _ := miner2.GetDestSplit().GetByID(contractID)
	destSplit3, _ := miner3.GetDestSplit().GetByID(contractID)

	require.Zero(t, globalScheduler.checkRedZones(destSplit1.Fraction), "shouldn't be in red zone")
	require.Zero(t, globalScheduler.checkRedZones(destSplit2.Fraction), "shouldn't be in red zone")
	require.Zero(t, globalScheduler.checkRedZones(destSplit3.Fraction), "shouldn't be in red zone")

	actualGHS := int(float64(miner1.GetHashRateGHS())*destSplit1.Fraction +
		float64(miner2.GetHashRateGHS())*destSplit2.Fraction +
		float64(miner3.GetHashRateGHS())*destSplit3.Fraction)

	require.InEpsilon(t, newGHS, actualGHS, 0.1, "HR should be accurate")
}

func TestDecrAllocation(t *testing.T) {
	newGHS := 8000
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalSchedulerV2(miners, &lib.LoggerMock{}, 0, 0, 0, 1)

	miner1, _ := miners.Load("1")
	miner2, _ := miners.Load("2")
	t.Log(miner1.GetDestSplit().GetByID(contractID))
	t.Log(miner2.GetDestSplit().GetByID(contractID))

	err := globalScheduler.update(contractID, newGHS, dest, nil)
	if err != nil {
		t.Fatal(err)
		return
	}

	miner1, _ = miners.Load("1")
	miner2, _ = miners.Load("2")

	destSplit1, _ := miner1.GetDestSplit().GetByID(contractID)
	destSplit2, _ := miner2.GetDestSplit().GetByID(contractID)

	if destSplit1.Fraction != 0 {
		t.Fatal("should totally remove 1st miner from allocation")
	}
	if destSplit2.Fraction != 0.4 {
		t.Fatal("should reduce 2nd miner allocation")
	}
}

func TestDecrAllocationRemoveMiner(t *testing.T) {
	newGHS := 6000
	dest, _ := lib.ParseDest("stratum+tcp://user:pwd@host.com:3333")
	contractID := "test-contract"

	miners := CreateMockMinerCollection(contractID, dest)
	globalScheduler := NewGlobalSchedulerV2(miners, &lib.LoggerMock{}, 0, 0, 0, 1)

	err := globalScheduler.update(contractID, newGHS, dest, nil)
	if err != nil {
		t.Fatal(err)
		return
	}

	miner1, _ := miners.Load("1")
	miner2, _ := miners.Load("2")

	splitItem1, ok1 := miner1.GetDestSplit().GetByID(contractID)
	splitItem2, ok2 := miner2.GetDestSplit().GetByID(contractID)

	if ok1 {
		fmt.Println(splitItem1)
		t.Fatal("should remove miner which was the least allocated for the contract")
	}
	if !ok2 {
		t.Fatal("should not remove second miner")
	}
	if splitItem2.Fraction != 0.3 {
		t.Fatal("should not alter allocation of the second miner")
	}
	if !miner1.GetDestSplit().IsEmpty() {
		t.Fatal("least allocated miner split should be empty")
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

	globalScheduler := NewGlobalSchedulerV2(miners, &lib.LoggerMock{}, 0, 0, 0, 1)
	snapshot := globalScheduler.GetMinerSnapshot()

	if _, ok := snapshot.Miner("2"); ok {
		t.Fatal("should filter out recently connected miner")
	}
	if _, ok := snapshot.Miner("1"); !ok {
		t.Fatal("a miner 1 should be available")
	}
}

func TestTryReduceMiners(t *testing.T) {
	gs := NewGlobalSchedulerV2(nil, lib.NewTestLogger(), 3, 5, 0.1, 1)
	col := data.NewAllocCollection()
	col.Add("miner-1", &data.AllocItem{
		MinerID:    "miner-1",
		ContractID: "contract",
		Fraction:   0.5,
		TotalGHS:   10000,
	})
	col.Add("miner-2", &data.AllocItem{
		MinerID:    "miner-2",
		ContractID: "contract",
		Fraction:   0.3,
		TotalGHS:   10000,
	})
	col.Add("miner-3", &data.AllocItem{
		MinerID:    "miner-3",
		ContractID: "contract",
		Fraction:   0.1,
		TotalGHS:   10000,
	})

	newCol := gs.tryReduceMiners(col)
	require.Equal(t, 2, newCol.GetZeroAllocatedCount(), "expected miners to be reduced")
	require.Equal(t, 9000, newCol.GetAllocatedGHS(), "collection hashrate should not change")

	item, _ := newCol.Get("miner-1")
	require.Equal(t, 0.9, item.Fraction, "incorrect fraction")
	require.Equal(t, 10000, item.TotalGHS, "incorrect totalGHS")
	require.Equal(t, "contract", item.ContractID, "incorrect contract ID")
}

func TestTryReduceMiners2(t *testing.T) {
	gs := NewGlobalSchedulerV2(nil, lib.NewTestLogger(), 3, 5, 0.1, 1)
	col := data.NewAllocCollection()
	col.Add("miner-1", &data.AllocItem{
		MinerID:    "miner-1",
		ContractID: "contract",
		Fraction:   0.33,
		TotalGHS:   94936,
	})
	col.Add("miner-2", &data.AllocItem{
		MinerID:    "miner-2",
		ContractID: "contract",
		Fraction:   0.32,
		TotalGHS:   116675,
	})
	col.Add("miner-3", &data.AllocItem{
		MinerID:    "miner-3",
		ContractID: "contract",
		Fraction:   0.33,
		TotalGHS:   96000,
	})
	col.Add("miner-4", &data.AllocItem{
		MinerID:    "miner-4",
		ContractID: "contract",
		Fraction:   1.0,
		TotalGHS:   103970,
	})

	newCol := gs.tryReduceMiners(col)

	require.Equal(t, 2, newCol.GetZeroAllocatedCount(), "expected miners to be reduced")
	require.Equal(t, newCol.GetAllocatedGHS(), col.GetAllocatedGHS(), "total hashrate is different")
}

func TestTryReduceMinersNotReduced(t *testing.T) {
	gs := NewGlobalSchedulerV2(nil, lib.NewTestLogger(), 3, 5, 0.1, 1)
	col := data.NewAllocCollection()
	col.Add("miner-1", &data.AllocItem{
		MinerID:    "miner-1",
		ContractID: "contract",
		Fraction:   0.9,
		TotalGHS:   10000,
	})
	col.Add("miner-2", &data.AllocItem{
		MinerID:    "miner-2",
		ContractID: "contract",
		Fraction:   0.3,
		TotalGHS:   10000,
	})

	newCol := gs.tryReduceMiners(col)
	require.Equal(t, 2, newCol.Len(), "expected miners to be not reduced")

	for key, newItem := range newCol.GetItems() {
		oldItem, ok := col.Get(key)
		require.Equal(t, true, ok)
		require.Equal(t, oldItem.Fraction, newItem.Fraction, "fraction")
		require.Equal(t, oldItem.TotalGHS, newItem.TotalGHS, "totalGHS")
		require.Equal(t, oldItem.ContractID, newItem.ContractID, "contractID")
		require.Equal(t, oldItem.MinerID, newItem.MinerID, "minerID")
	}
}

func TestTryAdjustRedZones(t *testing.T) {
	gs := NewGlobalSchedulerV2(nil, lib.NewTestLogger(), 2, 7, 0.1, 1)
	col := data.NewAllocCollection()
	col.Add("miner-1", &data.AllocItem{MinerID: "miner-1", ContractID: "contract", Fraction: 0.0, TotalGHS: 54030})
	col.Add("miner-2", &data.AllocItem{MinerID: "miner-2", ContractID: "contract", Fraction: 1.0, TotalGHS: 79941})
	col.Add("miner-3", &data.AllocItem{MinerID: "miner-3", ContractID: "contract", Fraction: 1.0, TotalGHS: 73335})
	col.Add("miner-4", &data.AllocItem{MinerID: "miner-4", ContractID: "contract", Fraction: 0.58, TotalGHS: 89324})
	col.Add("miner-5", &data.AllocItem{MinerID: "miner-5", ContractID: "contract", Fraction: 0.0, TotalGHS: 80706})

	gs.tryAdjustRedZones(col, nil)

	for _, item := range col.GetItems() {
		require.Equal(t, 0, gs.checkRedZones(item.Fraction), "shouldn't go into red zone")
	}
	//todo check if fractions did not change
}

func TestTryAdjustRedZones2(t *testing.T) {
	gs := NewGlobalSchedulerV2(nil, lib.NewTestLogger(), 2, 7, 0.1, 1)
	col := data.NewAllocCollection()
	col.Add("miner-1", &data.AllocItem{MinerID: "miner-1", ContractID: "contract", Fraction: 0.46, TotalGHS: 138347})
	col.Add("miner-2", &data.AllocItem{MinerID: "miner-2", ContractID: "contract", Fraction: 1.0, TotalGHS: 142011})

	gs.tryAdjustRedZones(col, nil)

	for _, item := range col.GetItems() {
		require.Equal(t, 0, gs.checkRedZones(item.Fraction), "shouldn't go into red zone")
	}
	//todo check if fractions did not change
}

func TestTryAdjustRedZonesLeft(t *testing.T) {
	gs := NewGlobalSchedulerV2(nil, lib.NewTestLogger(), 2, 7, 0.1, 1)
	col := data.NewAllocCollection()
	col.Add("miner-1", &data.AllocItem{
		MinerID:    "miner-1",
		ContractID: "contract",
		Fraction:   0.6,
		TotalGHS:   10000,
	})
	col.Add("miner-2", &data.AllocItem{
		MinerID:    "miner-2",
		ContractID: "contract",
		Fraction:   0.1,
		TotalGHS:   10000,
	})

	gs.tryAdjustRedZones(col, nil)

	for _, item := range col.GetItems() {
		require.Equal(t, 0, gs.checkRedZones(item.Fraction), "shouldn't go into red zone")
	}
	require.Equal(t, true, lib.AlmostEqual(col.GetAllocatedGHS(), 7000, 0.01), "total hashrate shouldn't change")
}

func TestTryAdjustRedZonesLeftNotPossible(t *testing.T) {
	gs := NewGlobalSchedulerV2(nil, lib.NewTestLogger(), 2, 7, 0.1, 1)
	col := data.NewAllocCollection()
	col.Add("miner-1", &data.AllocItem{
		MinerID:    "miner-1",
		ContractID: "contract",
		Fraction:   0.7,
		TotalGHS:   5000,
	})
	col.Add("miner-2", &data.AllocItem{
		MinerID:    "miner-2",
		ContractID: "contract",
		Fraction:   0.1,
		TotalGHS:   20000,
	})

	gs.tryAdjustRedZones(col, nil)

	m1, _ := col.Get("miner-1")
	m2, _ := col.Get("miner-2")

	require.Equal(t, 0.7, m1.Fraction)
	require.Equal(t, 0.1, m2.Fraction)
}

func TestTryAdjustRedZonesRightWFreeMiner(t *testing.T) {
	gs := NewGlobalSchedulerV2(nil, lib.NewTestLogger(), 2, 7, 0.1, 1)

	snap := data.NewAllocSnap()
	snap.SetMiner("miner-2", 10000)
	snap.Set("miner-1", "contract", 0.88, 10000)

	col, _ := snap.Contract("contract")

	gs.tryAdjustRedZones(col, &snap)

	for _, item := range col.GetItems() {
		require.Equal(t, 0, gs.checkRedZones(item.Fraction), "shouldn't go into red zone")
	}

	require.Equal(t, true, lib.AlmostEqual(col.GetAllocatedGHS(), 8800, 0.01), "hashrate shouldn't change")
}

func TestTryAdjustRedZonesRightWBusyMiner(t *testing.T) {
	gs := NewGlobalSchedulerV2(nil, lib.NewTestLogger(), 2, 7, 0.1, 1)

	snap := data.NewAllocSnap()
	snap.Set("miner-2", "contract", 0.3, 10000)
	snap.Set("miner-1", "contract", 0.88, 10000)

	col, _ := snap.Contract("contract")

	gs.tryAdjustRedZones(col, &snap)

	for _, item := range col.GetItems() {
		require.Equal(t, 0, gs.checkRedZones(item.Fraction), "shouldn't go into red zone")
	}

	require.Equal(t, true, lib.AlmostEqual(col.GetAllocatedGHS(), 11800, 0.01), "hashrate shouldn't change")
}

func TestTryAdjustRedZonesRightNotPossible(t *testing.T) {
	gs := NewGlobalSchedulerV2(nil, lib.NewTestLogger(), 2, 7, 0.1, 1)

	snap := data.NewAllocSnap()
	snap.SetMiner("miner-2", 1000)
	snap.Set("miner-1", "contract", 0.88, 10000)

	col, _ := snap.Contract("contract")

	gs.tryAdjustRedZones(col, &snap)

	m1, _ := col.Get("miner-1")
	_, m2ok := col.Get("miner-2")

	require.Equal(t, 0.88, m1.Fraction, "should do nothing")
	require.False(t, m2ok, "should do nothing")
}

func TestFindMidpointSplitWRedzones(t *testing.T) {
	minFraction, maxFraction := 0.3, 0.7

	var tests = []struct {
		totalHR1 int
		totalHR2 int
		targetHR int
	}{
		{10000, 10000, 11000},
		{10000, 20000, 11000},
		{10000, 30000, 15000},
		{20000, 15000, 21000},
		{10000, 5000, 10100},
		{20000, 30000, 21000},
	}

	for _, tt := range tests {
		t.Run("split correctly", func(t *testing.T) {
			f1, f2, ok := FindMidpointSplitWRedzones(minFraction, maxFraction, tt.totalHR1, tt.totalHR2, tt.targetHR)

			require.Equal(t, true, ok, "must be solvable for these values")
			require.InEpsilon(t, float64(tt.totalHR1)*f1+float64(tt.totalHR2)*f2, tt.targetHR, 0.01, "hashrate should match target")
			require.Truef(t, minFraction < f1 && f1 < maxFraction, "should not be in red zone %.3d", f1)
			require.Truef(t, minFraction < f2 && f2 < maxFraction, "should not be in red zone %.3d", f2)
		})
	}
}

func TestUpdateChangeDest(t *testing.T) {
	dest1 := lib.MustParseDest("stratum+tcp://user:pwd@host.com:3333")
	dest2 := lib.MustParseDest("stratum+tcp://user:pwd@hostchanged.com:3333")
	destDefault := lib.MustParseDest("stratum+tcp://default:pwd@host.com:3333")
	contractID := "contract"
	hrGHS := 10000
	minTime := 2 * time.Minute
	maxTime := 5 * time.Minute
	vettingPeriod := time.Second * 10

	miner1 := &protocol.MinerModelMock{
		ID:          "1",
		Dest:        destDefault,
		HashrateGHS: 10000,
		ConnectedAt: time.Now().Add(-time.Hour),
	}
	miner2 := &protocol.MinerModelMock{
		ID:          "2",
		Dest:        destDefault,
		HashrateGHS: 20000,
		ConnectedAt: time.Now(),
	}

	l, _ := lib.NewDevelopmentLogger("info", false, false, false)
	log := l.Sugar()

	scheduler1 := miner.NewOnDemandMinerScheduler(miner1, miner.NewDestSplit(), log, destDefault, vettingPeriod, minTime, maxTime)
	scheduler2 := miner.NewOnDemandMinerScheduler(miner2, miner.NewDestSplit(), log, destDefault, vettingPeriod, minTime, maxTime)

	go func() {
		_ = scheduler1.Run(context.Background())
	}()
	go func() {
		_ = scheduler2.Run(context.Background())
	}()

	miners := miner.NewMinerCollection()
	miners.Store(scheduler1)
	miners.Store(scheduler2)

	gs := NewGlobalSchedulerV2(miners, log, minTime, maxTime, 0.1, 1)

	err := gs.update(contractID, hrGHS, dest1, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = gs.update(contractID, hrGHS, dest2, nil)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	// search if change dest was called
	found := false
	for _, arg := range miner1.ChangeDestCalledWith {
		if arg.String() == dest2.String() {
			found = true
		}
	}
	for _, arg := range miner2.ChangeDestCalledWith {
		if arg.String() == dest2.String() {
			found = true
		}
	}

	assert.True(t, found)
}
