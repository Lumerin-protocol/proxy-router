package interfaces

type IRoutableStreamFullfillment interface {
	GetPercentage() float64
	GetTotalHashrate() uint64
	GetSourceID() string

	GetHashrate() uint64
	SetHashrate(uint64)
}
