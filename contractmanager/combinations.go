package contractmanager

import "fmt"

func FindCombinations(list HashrateList, targetHashrate int) HashrateList {

	combination, delta := FindClosestMinerCombination(list, targetHashrate)
	fmt.Printf("target %d delta %d", targetHashrate, delta)

	if combination.Len() == 0 {
		return combination
	}

	// partially allocate last miner
	// TODO: fix case when delta is larger than hashrate of one of the miner
	combination[combination.Len()-1].Hashrate = combination[combination.Len()-1].Hashrate - delta

	return combination
}

func FindClosestMinerCombination(list HashrateList, target int) (lst HashrateList, delta int) {
	hashrates := make([]int, list.Len())
	for i, v := range list {
		hashrates[i] = v.Hashrate
	}
	indexes, delta := ClosestSubsetSumRGLI(hashrates, target)

	res := make(HashrateList, len(indexes))
	for i, v := range indexes {
		res[i] = list[v]
	}
	return res, -delta // invert delta as it is always less than 0 to simplify usage
}
