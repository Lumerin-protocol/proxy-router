package contractmanager

func FindCombinations(list HashrateList, targetHashrate uint64) HashrateList {
	combination, delta := FindClosestMinerCombination(list, targetHashrate)
	// last element should always be larger than delta
	combination[combination.Len()-1].SetHashrate(combination[combination.Len()-1].GetHashrate() - uint64(delta))
	return combination
}

func FindClosestMinerCombination(list HashrateList, target uint64) (lst HashrateList, delta int) {
	hashrates := make([]uint64, list.Len())
	for i, v := range list {
		hashrates[i] = v.GetHashrate()
	}
	indexes, delta := ClosestSubsetSum(hashrates, target)

	res := make(HashrateList, len(indexes))
	for i, v := range indexes {
		res[i] = list[v]
	}
	return res, delta
}
