package interfaces

import "time"

type IHashrate interface {
	OnSubmit(diff int64)
}

type Hashrate interface {
	GetHashrateGHS() int
	GetHashrate5minAvgGHS() int
	GetHashrate30minAvgGHS() int
	GetHashrate1hAvgGHS() int
	GetHashrateAvgGHSCustom(avg time.Duration) (hrGHS int, ok bool)
}
