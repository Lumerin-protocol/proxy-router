package contractmanager

import (
	"fmt"
	"math"
	"math/rand"
)

// RGLI_TRIALS is number of attempts to pick the best combination, recommended value from 10 to 50
const RGLI_TRIALS = 10

// ClosestSubsetSumRGLI implements approgimate RGLI algo to solve closest subset sum problem
//  Sources:
// https://rpubs.com/aviadt/subset-sum
// https://web.stevens.edu/algebraic/Files/SubsetSum/przydatek99fast.pdf
func ClosestSubsetSumRGLI(a []int, B int) (numIndexes []int, dlt int) {
	n := len(a)
	x_best := rep(0, n)
	x_best_delta := math.MaxInt

	for i := 0; i < RGLI_TRIALS; i++ {
		x := trial(a, B)

		x_delta := delta(x, a, B)
		if x_delta < x_best_delta {
			fmt.Printf("Current solution better than best so far. Updating. %v\n", x)
			x_best = x
			x_best_delta = x_delta
		}

		if x_best_delta == 0 {
			fmt.Printf("Found perfect solution! %+v\n", x)
			break // quit the outer for loop
		}
	}

	return whichInd(x_best, 1), x_best_delta
}

func trial(a []int, B int) []int {
	n := len(a)

	// first phase: randomized selection
	x := rep(0, n)
	for _, v := range shuffle(inc(n)) {
		// go over elements in random order
		if a[v] <= delta(x, a, B) {
			x[v] = 1
		}
	}

	// second phase: local improvement
	I := whichInd(x, 1)
	for _, i := range shuffle(I) {
		// for _, i := range I { // you can remove shuffle to have more predictable result in tests
		// in random order
		delta_x := delta(x, a, B)
		if delta_x == 0 {
			break // quit the inner for loop
		}

		// find potential elements to improve current solution
		// T_idx := which(x == 0 & a-a[i] > 0 & a-a[i] <= delta_x)
		notIncludedMask := isEqualArrMask(x, 0)
		largerThanCurMask := isGreaterArrMask(subArr(a, a[i]), 0)
		smallerOrEqToTargetMask := isSubOrEqArrMask(subArr(a, a[i]), delta_x)

		fitsMask := boolAndMask(boolAndMask(notIncludedMask, largerThanCurMask), smallerOrEqToTargetMask)

		T_idx := whichInd(fitsMask, 1)
		if len(T_idx) > 0 {
			fmt.Printf("Local update %v\n", fitsMask)

			// k := T_idx[which.max(a[T_idx])]
			maxItemInd := maxArrInd(pickArr(a, T_idx))
			k := T_idx[maxItemInd]
			x[k] = 1
			x[i] = 0
		}
	}

	return x
}

// delta applies valueMask to allValues, sums them, and return difference with targetSum
func delta(valueMask []int, allValues []int, targetSum int) int {
	var sum int = 0
	for i := 0; i < len(valueMask); i++ {
		sum += valueMask[i] * allValues[i]
	}
	return targetSum - sum
}

// maxArrInd returns index of the max element in array
func maxArrInd(arr []int) int {
	res := math.MinInt
	for i, v := range arr {
		if v > res {
			res = i
		}
	}
	return res
}

// pickArr returns new array with elements, than are under the ind indexes in source array arr
func pickArr(arr []int, ind []int) []int {
	res := []int{}
	for _, v := range ind {
		res = append(res, arr[v])
	}
	return res
}

// boolAndMask performs positional logical AND for its arguments
func boolAndMask(a []int, b []int) []int {
	res := make([]int, len(a))
	for i := 0; i < len(a); i++ {
		r := (a[i] == 1) && (b[i] == 1)
		if r {
			res[i] = 1
		} else {
			res[i] = 0
		}
	}
	return res
}

func isSubOrEqArrMask(a []int, val int) []int {
	res := make([]int, len(a))
	for i, v := range a {
		if v <= val {
			res[i] = 1
		} else {
			res[i] = 0
		}
	}
	return res
}

func isGreaterArrMask(a []int, val int) []int {
	res := make([]int, len(a))
	for i, v := range a {
		if v > val {
			res[i] = 1
		} else {
			res[i] = 0
		}
	}
	return res
}

// subArr subtracts val from each element of array, returns new array
func subArr(a []int, val int) []int {
	res := make([]int, len(a))
	for i, v := range a {
		res[i] = v - val
	}
	return res
}

// isEqualArrMask return mask where true means that array item is equal to target value
func isEqualArrMask(x []int, val int) []int {
	res := make([]int, len(x))
	for i, v := range x {
		if v == val {
			res[i] = 1
		} else {
			res[i] = 0
		}
	}
	return res
}

// whichInd return indexes of val in x array
func whichInd(x []int, val int) []int {
	res := []int{}
	for i, v := range x {
		if v == val {
			res = append(res, i)
		}
	}
	return res
}

// rep creates an array of length n with all values v
func rep(v int, n int) []int {
	res := make([]int, n)
	for i := 0; i < n; i++ {
		res[i] = v
	}
	return res
}

// inc creates an array of length n with incrementing values from 0 to n-1
func inc(n int) []int {
	res := make([]int, n)
	for i := 0; i < n; i++ {
		res[i] = i
	}
	return res
}

// shuffle returns new array where all elements are in random order
func shuffle(a []int) []int {
	//	rand.Seed(time.Now().UnixNano())
	dest := make([]int, len(a))
	copy(dest, a)
	rand.Shuffle(len(dest), func(i, j int) { dest[i], dest[j] = dest[j], dest[i] })
	return dest
}
