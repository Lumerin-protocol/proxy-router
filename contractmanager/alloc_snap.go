package contractmanager

import (
	"bytes"
	"fmt"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/miner"
	"golang.org/x/exp/slices"
)

type AllocItem struct {
	MinerID    string
	ContractID string
	Fraction   float64
	TotalGHS   int
}

func (m *AllocItem) AllocatedGHS() int {
	return int(float64(m.TotalGHS) * m.Fraction)
}
func (m *AllocItem) AvailableGHS() int {
	return int(float64(m.TotalGHS) * m.AvailableFraction())
}
func (m *AllocItem) AvailableFraction() float64 {
	return 1 - m.Fraction
}

type AllocCollection map[string]*AllocItem

func (m AllocCollection) SortByAllocatedGHS() []AllocItem {
	items := make([]AllocItem, len(m))
	i := 0
	for _, item := range m {
		items[i] = *item
		i++
	}
	slices.SortStableFunc(items, func(a, b AllocItem) bool {
		return a.AllocatedGHS() < b.AllocatedGHS()
	})
	return items
}

type AllocSnap struct {
	contractIDMinerIDMap map[string]AllocCollection
	minerIDcontractIDMap map[string]AllocCollection
}

func NewAllocSnap() AllocSnap {
	return AllocSnap{
		contractIDMinerIDMap: make(map[string]AllocCollection),
		minerIDcontractIDMap: make(map[string]AllocCollection),
	}
}

func (m *AllocSnap) Set(minerID string, contractID string, fraction float64, totalGHS int) {
	item := &AllocItem{
		MinerID:    minerID,
		ContractID: contractID,
		Fraction:   fraction,
		TotalGHS:   totalGHS,
	}
	_, ok := m.contractIDMinerIDMap[contractID]
	if !ok {
		m.contractIDMinerIDMap[contractID] = make(AllocCollection)
	}
	m.contractIDMinerIDMap[contractID][minerID] = item

	_, ok = m.minerIDcontractIDMap[minerID]
	if !ok {
		m.minerIDcontractIDMap[minerID] = make(AllocCollection)
	}
	m.minerIDcontractIDMap[minerID][contractID] = item
}

func (m *AllocSnap) Get(minerID string, contractID string) (AllocItem, bool) {
	contractIDMap, ok := m.minerIDcontractIDMap[minerID]
	if !ok {
		return AllocItem{}, false
	}
	item, ok := contractIDMap[contractID]
	if !ok {
		return AllocItem{}, false
	}
	return *item, true
}

func (m *AllocSnap) Miner(minerID string) (AllocCollection, bool) {
	res, ok := m.minerIDcontractIDMap[minerID]
	return res, ok
}

func (m *AllocSnap) Contract(contractID string) (AllocCollection, bool) {
	res, ok := m.contractIDMinerIDMap[contractID]
	return res, ok
}

func (m *AllocSnap) String() string {
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "\nContractID\tMinerID\tFraction\tTotalGHS\n")
	for _, item := range m.contractIDMinerIDMap {
		for _, alloc := range item {
			fmt.Fprintf(b, "%s\t%s\t%.2f\t%d\n", alloc.ContractID, alloc.MinerID, alloc.Fraction, alloc.TotalGHS)
		}
	}
	return b.String()
}

func CreateMinerSnapshot(minerCollection interfaces.ICollection[miner.MinerScheduler]) AllocSnap {
	snapshot := NewAllocSnap()
	minerCollection.Range(func(miner miner.MinerScheduler) bool {
		for _, splitItem := range miner.GetDestSplit().Iter() {
			snapshot.Set(miner.GetID(), splitItem.ID, splitItem.Percentage, miner.GetHashRateGHS())
		}
		return true
	})
	return snapshot
}
