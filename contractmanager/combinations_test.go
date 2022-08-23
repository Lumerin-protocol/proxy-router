package contractmanager

import (
	"fmt"
	"math/rand"
	"testing"
)

// func TestCombinations(t *testing.T) {
// 	t.Skip()
// 	m := MinerList{
// 		MinerHashrate{MinerID: "1", Hashrate: 100, Percentage: 0.1},
// 		MinerHashrate{MinerID: "2", Hashrate: 200, Percentage: 0.2},
// 		MinerHashrate{MinerID: "3", Hashrate: 300, Percentage: 1},
// 		MinerHashrate{MinerID: "4", Hashrate: 400, Percentage: 0.4},
// 	}
// 	subsets := findSubsets(m, 310, 0.1)
// 	for _, v := range subsets {
// 		fmt.Printf("%+v\n", v)
// 	}
// 	fmt.Println("")
// 	comb, _ := bestCombination(subsets, 310)
// 	fmt.Printf("%+v\n", comb)
// }

func TestCombinationsv2(t *testing.T) {
	t.Skip()
	arr := []int{400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400}
	res, delta := ClosestSubsetSum(arr, 2000)
	fmt.Printf("%+v === %d\n", res, delta)
}

func TestCombinationsv3(t *testing.T) {
	// t.Skip()
	size := 10
	randNums := make([]int, size)
	total := 0

	for i := 0; i < size; i++ {
		r := rand.Intn(1000)
		randNums[i] = r
		total += r
	}

	res, delta := ClosestSubsetSumRGLI(randNums, total/2)
	fmt.Printf("%+v\n", randNums)
	fmt.Printf("%+v === %d total %d\n", res, delta, total)
}

func TestCombinationsv3Larger(t *testing.T) {
	t.Skip()

	randNums := []int{400, 200, 100, 50, 400, 200, 250}

	res, delta := ClosestSubsetSumRGLI(randNums, 300)
	fmt.Printf("%+v === %d\n", res, delta)
}
