package contractmanager

import (
	"fmt"
	"testing"
)

func TestCombinations(t *testing.T) {
	t.Skip()
	m := MinerList{
		MinerHashrate{MinerID: "1", Hashrate: 100, Percentage: 0.1},
		MinerHashrate{MinerID: "2", Hashrate: 200, Percentage: 0.2},
		MinerHashrate{MinerID: "3", Hashrate: 300, Percentage: 1},
		MinerHashrate{MinerID: "4", Hashrate: 400, Percentage: 0.4},
	}
	subsets := findSubsets(m, 310, 0.1)
	for _, v := range subsets {
		fmt.Printf("%+v\n", v)
	}
	fmt.Println("")
	comb, _ := bestCombination(subsets, 310)
	fmt.Printf("%+v\n", comb)
}

func TestCombinationsv2(t *testing.T) {
	arr := []uint64{400, 300, 300, 300, 400}
	res, delta := MinAbsDifferenceArr(arr, 699)
	fmt.Printf("%+v === %d\n", res, delta)
}
