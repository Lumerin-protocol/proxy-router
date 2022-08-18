package contractmanager

import "math"

const (
	MIN_SLICE = 0.1
)

// find subsets of list of miners whose hashrate sum equal the target hashrate
func findSubsets(sortedMiners MinerList, targetHashrate uint64, hashrateTolerance float64) (minerCombinations []MinerList) {
	// Calculate total number of subsets
	tot := math.Pow(2, float64(sortedMiners.Len()))
	MAX := uint64(float64(targetHashrate) * (1 + hashrateTolerance))
	minerCombinationsSums := []uint64{}

	for i := 0; i < int(tot); i++ {
		m, s := sumSubsets(sortedMiners, i, targetHashrate, hashrateTolerance)
		if m != nil {
			minerCombinations = append(minerCombinations, m)
			minerCombinationsSums = append(minerCombinationsSums, s)
		}
	}

	if len(minerCombinations) == 0 {
		return []MinerList{}
	}

	for i, m := range minerCombinations {
		if minerCombinationsSums[i] > MAX { // need to slice miner
			sumPrev := minerCombinationsSums[i] - m[len(m)-1].Hashrate
			unslicedHashrate := m[len(m)-1].Hashrate
			slicedHashrate := targetHashrate - sumPrev
			if float64(slicedHashrate)/float64(unslicedHashrate) < MIN_SLICE {
				m[len(m)-1].Hashrate = uint64(float64(m[len(m)-1].Hashrate) * MIN_SLICE)
				m[len(m)-1].Percentage = MIN_SLICE
			} else {
				m[len(m)-1].Hashrate = slicedHashrate
				m[len(m)-1].Percentage = float64(slicedHashrate) / float64(unslicedHashrate)
			}
		}
	}

	return minerCombinations
}

func sumSubsets(sortedMiners MinerList, n int, targetHashrate uint64, hashrateTolerance float64) (m MinerList, sum uint64) {
	// Create new array with size equal to sorted miners array to create binary array as per n(decimal number)
	x := make([]int, sortedMiners.Len())
	j := sortedMiners.Len() - 1

	// Convert the array into binary array
	for n > 0 {
		x[j] = n % 2
		n = n / 2
		j--
	}

	var sumPrev uint64 = 0 // only return subsets where hashrate overflow is caused by 1 miner

	// Calculate the sum of this subset
	for i := range sortedMiners {
		if x[i] == 1 {
			sum += sortedMiners[i].Hashrate
		}
		if i == len(sortedMiners)-1 {
			sumPrev = sum - sortedMiners[i].Hashrate
		}
	}

	MIN := uint64(float64(targetHashrate) * (1 - hashrateTolerance))

	// if sum is within target hashrate bounds, subset was found
	if sum >= MIN && sumPrev < MIN {
		for i := range sortedMiners {
			if x[i] == 1 {
				m = append(m, sortedMiners[i])
			}
		}
		return m, sum
	}

	return nil, 0
}

func bestCombination(minerCombinations []MinerList, targetHashrate int) (MinerList, int) {
	hashrates := make([]uint64, len(minerCombinations))
	numMiners := make([]int, len(minerCombinations))

	// find hashrate and number of miners in each combination
	for i := range minerCombinations {
		miners := minerCombinations[i]
		var totalHashRate uint64 = 0
		num := 0
		for j := range miners {
			if j == len(miners)-1 {
				totalHashRate += uint64(float64(miners[j].Hashrate) * miners[j].Percentage)
			} else {
				totalHashRate += miners[j].Hashrate
			}
			num++
		}

		hashrates[i] = totalHashRate
		numMiners[i] = num
	}

	// find combination closest to target hashrate
	index := 0
	for i := range hashrates {
		res1 := math.Abs(float64(targetHashrate) - float64(hashrates[index]))
		res2 := math.Abs(float64(targetHashrate) - float64(hashrates[i]))
		if res1 > res2 {
			index = i
		}
	}

	// if duplicate exists choose the one with the least number of miners
	newIndex := index
	for i := range hashrates {
		if hashrates[i] == hashrates[index] && numMiners[i] < numMiners[newIndex] {
			newIndex = i
		}
	}

	return minerCombinations[newIndex], newIndex
}
