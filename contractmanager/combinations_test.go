package contractmanager

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCombinationsv2(t *testing.T) {
	t.Skip()
	arr := []int{400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400, 300, 300, 100, 100, 300, 500, 600, 700, 800, 400}
	res, delta := ClosestSubsetSum(arr, 200000)
	fmt.Printf("%+v === %d\n", res, delta)
}

func TestCombinationsv3(t *testing.T) {
	t.Skip()
	size := 10
	randNums := make([]int, size)
	total := 0

	for i := 0; i < size; i++ {
		r := rand.Intn(1000)
		randNums[i] = r
		total += r
	}

	target := 50
	res, delta := ClosestSubsetSumRGLI(randNums, target)
	fmt.Printf("input: %+v\n", randNums)
	fmt.Printf("output: %+v\n delta: %d target: %d total: %d\n", res, delta, target, total)
}

func TestCombinationsv3Overallocation(t *testing.T) {
	arr := []int{100, 100, 50, 50}
	_, delta := ClosestSubsetSumRGLI(arr, 180)
	assert.Equal(t, delta, -20, "should overallocate if possible")
}

func TestCombinationsv3NotEnoughHR(t *testing.T) {
	arr := []int{100, 100, 50, 50}
	res, delta := ClosestSubsetSumRGLI(arr, 350)
	assert.Equal(t, len(arr), len(res), "should use all elements if not enough hashrate")
	assert.Equal(t, delta, 50, "should return correct delta if not enough hashrate")
}

func TestCombinationsv3Larger(t *testing.T) {
	t.Skip()

	randNums := []int{400, 200, 100, 50, 400, 200, 250}

	res, delta := ClosestSubsetSumRGLI(randNums, 300)
	fmt.Printf("%+v === %d\n", res, delta)
}
