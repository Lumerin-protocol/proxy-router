package contractmanager

import (
	"sort"

	snap "gitlab.com/TitanInd/hashrouter/data"
)

// FindCombinations returns any number of miner splits that together have a target hashrate or more
// if delta is negative it means underallocation, if positive - overallocation
func FindCombinations(list *snap.AllocCollection, target int) (comb *snap.AllocCollection, delta int) {
	keys := make([]string, 0)
	for k, item := range list.GetItems() {
		// exclude miners with zero hashrate
		if item.TotalGHS > 0 {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)

	hashrates := make([]int, len(keys))
	for i, key := range keys {
		hashrates[i] = list.GetItems()[key].AllocatedGHS()
	}
	indexes, delta := ClosestSubsetSumRGLI(hashrates, target)

	res := snap.NewAllocCollection()

	for _, index := range indexes {
		key := keys[index]
		res.Add(key, list.GetItems()[key])
	}

	return res, delta
}
