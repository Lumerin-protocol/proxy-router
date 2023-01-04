package lib

import (
	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Integer | constraints.Float
}

func AlmostEqual[T Number](a, b T, tolerance float64) bool {
	return float64(Abs(a-b))/float64(a) < tolerance
}

func Abs[T Number](a T) T {
	if a < 0 {
		return -a
	}
	return a
}
