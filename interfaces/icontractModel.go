package interfaces

type IContractModel interface {
	SetDestination(string)
	HasDestination() bool
	IsAvailable() bool
	MakeAvailable()
	GetAddress() string
	GetPromisedHashrateMin() uint64
	GetPrivateKey() string
	GetBuyerAddress() string
	GetCurrentNonce() uint64
	Save()
	Execute()
}
