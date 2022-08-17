package interfaces

type IContractModel interface {
	IBaseModel
	SetDestination(string)
	HasDestination() bool
	IsAvailable() bool
	MakeAvailable()
	GetAddress() string
	GetPromisedHashrateMin() uint64
	GetPrivateKey() string
	GetBuyerAddress() string
	SetBuyerAddress(buyer string)
	GetCurrentNonce() uint64
	Save() (IContractModel, error)
	Execute() (IContractModel, error)
	GetCloseOutType() uint
	TryRunningAt(dest string) (IContractModel, error)
}
