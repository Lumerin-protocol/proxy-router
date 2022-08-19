package contractmanager

type bitMask []bool

func NewBitMask(size int) bitMask {
	return make(bitMask, size)
}

func (b bitMask) Set(ind int, val bool) {
	b[ind] = val
}

func (b bitMask) Get(ind int) bool {
	return b[ind]
}

func (b bitMask) Which(val bool) []int {
	res := []int{}
	for i, v := range b {
		if v == val {
			res = append(res, i)
		}
	}
	return res
}
