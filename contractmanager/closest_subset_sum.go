package contractmanager

import (
	"math"
	"sort"
)

// ClosestSubsetSum finds the subset of elements in array sum of which is closest to goal
// The resulting sum is larger or equal to the goal
//  numIndexes - indexes of elements from incoming array that make the subset
//  delta - delta between actual value and expected
//
// Copy-pasted from: https://leetcode.com/problems/closest-subsequence-sum/discuss/2237271/Go-solution-with-explanation
func ClosestSubsetSum(numbers []int, goal int) (numIndexes []int, delta int) {
	nums := make([][2]int, len(numbers))
	for i, v := range numbers {
		nums[i] = [2]int{i, v}
	}
	mid := len(nums) / 2

	left := nums[:mid]
	right := nums[mid:]

	leftSum := getAllSumArr(left)
	rightSum := getAllSumArr(right)

	sort.Slice(leftSum, func(i, j int) bool {
		return sumIndexed(leftSum[i]) < sumIndexed(leftSum[j])
	})
	sort.Slice(rightSum, func(i, j int) bool {
		return sumIndexed(rightSum[i]) < sumIndexed(rightSum[j])
	})

	indexedRes, delt := getMinValArr(leftSum, rightSum, goal)

	indexes := make([]int, len(indexedRes))
	for i, v := range indexedRes {
		indexes[i] = v[0]
	}

	return indexes, delt
}

func getMinValArr(leftSum, rightSum [][][2]int, goal int) (valArr [][2]int, delta int) {
	var minSoFar int = math.MaxInt
	minSoFarArr := [][2]int{}

	i := 0
	j := len(rightSum) - 1

	for i < len(leftSum) && j >= 0 {
		leftItem := sumIndexed(leftSum[i])
		rightItem := sumIndexed(rightSum[j])
		sumx := leftItem + rightItem

		// closest either larger or smaller value
		// if minSoFar > abs(goal, sumx) {
		// 	minSoFar = abs(goal, sumx)
		// 	minSoFarArr = append(append([][2]int{}, leftSum[i]...), rightSum[j]...)
		// }

		// closest larger combination
		delta := sub(sumx, goal)
		if delta >= 0 && delta < minSoFar {
			minSoFar = delta
			minSoFarArr = append(append([][2]int{}, leftSum[i]...), rightSum[j]...)
		}

		if sumx < goal {
			i++
		} else if sumx > goal {
			j--
		} else {
			break
		}
	}

	return minSoFarArr, minSoFar
}

func getAllSumArr(nums [][2]int) [][][2]int {
	n := len(nums)

	var res [][][2]int

	var iter func(idx int, sumSoFar [][2]int)
	iter = func(idx int, sumSoFar [][2]int) {
		if idx == n {
			if len(sumSoFar) > 0 {
				res = append(res, sumSoFar)
			}
			return
		}

		iter(idx+1, sumSoFar)
		iter(idx+1, append(sumSoFar, nums[idx]))
	}

	iter(0, [][2]int{})

	return res
}

func sumIndexed(a [][2]int) int {
	var res int
	for _, v := range a {
		res += v[1]
	}
	return res
}

func sum(a []int) int {
	var res int
	for _, v := range a {
		res += v
	}
	return res
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(a, b int) int {
	if a < b {
		return b - a
	}
	return a - b
}

func sub(a, b int) int {
	// avoiding overflow
	if a < b {
		return -(b - a)
	}
	return (a - b)
}
