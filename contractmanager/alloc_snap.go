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

func (m *AllocItem) String() string {
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "\nContractID\tMinerID\tFraction\tTotalGHS\n")
	fmt.Fprintf(b, "%s\t%s\t%.2f\t%d\n", m.ContractID, m.MinerID, m.Fraction, m.TotalGHS)
	return b.String()
}

type AllocCollection map[string]*AllocItem

func (m AllocCollection) FilterFullyAvailable() AllocCollection {
	fullyAvailable := AllocCollection{}
	for key, item := range m {
		if item.Fraction == 1 {
			fullyAvailable[key] = item
		}
	}
	return fullyAvailable
}

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

func (m AllocCollection) String() string {
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "\nContractID\tMinerID\tFraction\tTotalGHS\n")
	for _, alloc := range m {
		fmt.Fprintf(b, "%s\t%s\t%.2f\t%d\n", alloc.ContractID, alloc.MinerID, alloc.Fraction, alloc.TotalGHS)
	}
	return b.String()
}

func (m AllocCollection) Len() int {
	return len(m)
}

func (m AllocCollection) IDs() []string {
	minerIDs := make([]string, m.Len())
	i := 0
	for _, item := range m {
		minerIDs[i] = item.MinerID
		i++
	}
	return minerIDs
}

func (m AllocCollection) GetUnallocatedGHS() (int, *AllocItem) {
	var allocatedFrac float64 = 0
	var allocItemAvailable *AllocItem
	var minerID string
	var totalGHS int

	for _, item := range m {
		allocatedFrac += item.Fraction
		minerID = item.MinerID
		totalGHS = item.TotalGHS
	}

	availableFrac := 1 - allocatedFrac
	allocItemAvailable = &AllocItem{
		MinerID:    minerID,
		ContractID: "",
		Fraction:   availableFrac,
		TotalGHS:   totalGHS,
	}

	return allocItemAvailable.AllocatedGHS(), allocItemAvailable
}

type AllocSnap struct {
	contractIDMinerIDMap map[string]AllocCollection
	minerIDcontractIDMap map[string]AllocCollection
	minerIDHashrateGHS   map[string]int
}

func NewAllocSnap() AllocSnap {
	return AllocSnap{
		contractIDMinerIDMap: make(map[string]AllocCollection),
		minerIDcontractIDMap: make(map[string]AllocCollection),
		minerIDHashrateGHS:   make(map[string]int),
	}
}

func (m *AllocSnap) Set(minerID string, contractID string, fraction float64, totalGHS int) {
	m.SetMiner(minerID, totalGHS)

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

func (m *AllocSnap) SetMiner(minerID string, hashrateGHS int) {
	_, ok := m.minerIDcontractIDMap[minerID]
	if !ok {
		m.minerIDcontractIDMap[minerID] = make(AllocCollection)
	}
	m.minerIDHashrateGHS[minerID] = hashrateGHS
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

func (s *AllocSnap) GetUnallocatedGHS() (int, AllocCollection) {
	var unallocatedHashrateGHS int = 0
	allocItemsAvailable := make(AllocCollection)

	for minerID, miner := range s.minerIDcontractIDMap {
		_, allocItem := miner.GetUnallocatedGHS()

		if allocItem.Fraction > 0 {
			item := &AllocItem{
				MinerID:    minerID,
				ContractID: "",
				Fraction:   allocItem.Fraction,
				TotalGHS:   s.minerIDHashrateGHS[minerID],
			}
			allocItemsAvailable[minerID] = item
			unallocatedHashrateGHS += item.AllocatedGHS()
		}
	}

	return unallocatedHashrateGHS, allocItemsAvailable
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
		hashrateGHS := miner.GetHashRateGHS()
		minerID := miner.GetID()

		snapshot.SetMiner(minerID, hashrateGHS)

		for _, splitItem := range miner.GetDestSplit().Iter() {
			snapshot.Set(minerID, splitItem.ID, splitItem.Percentage, hashrateGHS)
		}

		return true
	})
	return snapshot
}
