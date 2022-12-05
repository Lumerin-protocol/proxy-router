package interfaces

type IHashrate interface {
	OnSubmit(diff int64)
}

type Hashrate interface {
	GetHashrate5minAvgGHS() int
	GetHashrate30minAvgGHS() int
	GetHashrate1hAvgGHS() int
}
