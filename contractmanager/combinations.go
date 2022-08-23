package contractmanager

func FindCombinations(list HashrateList, targetHashrate int) HashrateList {
	combination, delta := FindClosestMinerCombination(list, targetHashrate)

	// partially allocate last miner
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
	return res, delta
}
