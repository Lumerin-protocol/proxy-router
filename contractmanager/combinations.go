package contractmanager

import (
	"fmt"
	"sort"
)

// FindCombinations returns any number of miner splits that together have a target hashrate or more
func FindCombinations(list *AllocCollection, targetHashrate int) (*AllocCollection, int) {

	combination, delta := FindClosestMinerCombination(list, targetHashrate)
	fmt.Printf("target %d delta %d", targetHashrate, delta)

	return combination, delta
}

func FindClosestMinerCombination(list *AllocCollection, target int) (lst *AllocCollection, delta int) {
	keys := make([]string, 0)
	for k := range list.GetItems() {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	hashrates := make([]int, len(list.GetItems()))
	for i, key := range keys {
		hashrates[i] = list.GetItems()[key].AllocatedGHS()
	}
	indexes, delta := ClosestSubsetSumRGLI(hashrates, target)

	res := &AllocCollection{items: make(map[string]*AllocItem, len(indexes))}

	for _, index := range indexes {
		key := keys[index]
		res.Add(key, list.GetItems()[key])
	}

	return res, -delta // invert delta as it is always less than 0 to simplify usage
}
