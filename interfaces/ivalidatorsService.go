package interfaces

type IValidatorsService interface {
	GetHashrate() (uint64, error)
}
