package interfaces

type IContractModel interface {
	SetDestination(string)
	HasDestination() bool
	IsAvailable() bool
	GetHexAddress() string
}
