package contractmanager

import (
	"fmt"
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
	arr := []int{400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800}
	res, delta := ClosestSubsetSum(arr, 2000)
	fmt.Printf("%+v === %d\n", res, delta)
}

func TestCombinationsv3(t *testing.T) {
	// arr := []int{800, 150, 10000, 10000, 175}
	arr := []int{400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800}

	res, delta := ClosestSubsetSumRGLI(arr, 1000)
	fmt.Printf("%+v === %d\n", res, delta)
}
