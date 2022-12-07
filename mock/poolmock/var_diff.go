package poolmock

import "math"

type VarDiff struct {
	step         int
	varDiffRange [2]int
}

func NewVarDiff(varDiffRange [2]int) *VarDiff {
	return &VarDiff{
		varDiffRange: varDiffRange,
	}
}

func (v *VarDiff) Inc() bool {
	if v.val(v.step+1) > v.varDiffRange[1] {
		return false
	}
	v.step++
	return true
}

func (v *VarDiff) Dec() bool {
	if v.step == 0 {
		return false
	}
	v.step--
	return true
}

func (v *VarDiff) Val() int {
	return v.val(v.step)
}

func (v *VarDiff) val(step int) int {
	diff := float64(v.varDiffRange[0]) * math.Pow(2, float64(step))
	return int(diff)
}
