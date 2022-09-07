package contractmanager

import (
	"math"
	"math/rand"
	"time"
)

// RGLI_TRIALS is number of attempts to pick the best combination, recommended value from 10 to 50
const RGLI_TRIALS = 50

// ClosestSubsetSumRGLI implements approgimate RGLI algo to solve closest subset sum problem
//
//		Added changes so the algo always returns value larger or equal to target
//	 Sources:
//
// https://rpubs.com/aviadt/subset-sum
// https://web.stevens.edu/algebraic/Files/SubsetSum/przydatek99fast.pdf
//  Delta should be <= 0 that means that sum is overallocated
func ClosestSubsetSumRGLI(arr []int, sum int) (numIndexes []int, dlt int) {
	if !SEED_CALLED {
		Seed(time.Now().UnixNano())
	}

	resBestMask := NewBitMask(len(arr))
	resBestDelta := math.MaxInt

	for i := 0; i < RGLI_TRIALS; i++ {
		resMask := trial(arr, sum)

		resDelta := delta(resMask, arr, sum)
		if absInt(resDelta) < absInt(resBestDelta) {
			resBestMask = resMask
			resBestDelta = resDelta
		}

		if resBestDelta == 0 {
			break
		}
	}

	return resBestMask.Which(true), resBestDelta
}

func trial(a []int, B int) bitMask {
	n := len(a)

	// first phase: randomized selection
	x := NewBitMask(n)
	for _, v := range shuffle(incRange(n)) {
		// go over elements in random order
		if delta(x, a, B) > 0 {
			x.Set(v, true)
		}
	}

	// second phase: local improvement
	for _, existingInd := range shuffle(x.Which(true)) { // you can remove shuffle to have more predictable result in tests
		delta_x := delta(x, a, B) // diff between actual sum and target
		if delta_x == 0 {
			break // quit the inner for loop
		}

		// find potential elements to improve current solution
		replacementInd := -1 // -1 means not found yet

		for j := 0; j < n; j++ {
			delta := a[j] - a[existingInd]
			if (!x.Get(j)) && absInt(delta+delta_x) < absInt(delta_x) {
				replacementInd = j
			}
		}

		if replacementInd >= 0 {
			x.Set(replacementInd, true)
			x.Set(existingInd, false)
		}
	}

	return x
}

// delta applies valueMask to allValues, sums them, and return difference with targetSum
func delta(valueMask bitMask, allValues []int, targetSum int) int {
	var sum int = 0
	for i := 0; i < len(valueMask); i++ {
		if valueMask.Get(i) {
			sum += allValues[i]
		}
	}
	return targetSum - sum
}

// incRange creates an array of length n with incrementing values from 0 to n-1
func incRange(n int) []int {
	res := make([]int, n)
	for i := 0; i < n; i++ {
		res[i] = i
	}
	return res
}

// shuffle returns new array where all elements are in random order
func shuffle(a []int) []int {
	dest := make([]int, len(a))
	copy(dest, a)
	rand.Shuffle(len(dest), func(i, j int) { dest[i], dest[j] = dest[j], dest[i] })
	return dest
}

func absInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

var SEED_CALLED bool = false // helper to ensure seed called only once

// seed seeds the global random with desired value (useful for tests)
func Seed(seed int64) {
	rand.Seed(seed)
	SEED_CALLED = true
}
