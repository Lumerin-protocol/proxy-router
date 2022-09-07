package contractmanager

import "fmt"

func FindCombinations(list HashrateList, targetHashrate int) HashrateList {

	combination, delta := FindClosestMinerCombination(list, targetHashrate)
	fmt.Printf("target %d delta %d", targetHashrate, delta)

	// partially allocate last miner
	combination[combination.Len()-1].Hashrate = combination[combination.Len()-1].Hashrate - delta

	fmt.Printf("\n\n")
	for _, item := range combination {
		fmt.Printf("id %s HR %d TOTAL HR %d PERCENT %.3f", item.MinerID, item.Hashrate, item.TotalHashrate, item.GetPercentage())
	}
	fmt.Printf("\n\n")

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
