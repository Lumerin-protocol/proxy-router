package interfaces

type ISellerContractModel interface {
	IBaseModel
	SetDestination(string) error
	IsAvailable() bool
	MakeAvailable()
	GetAddress() string
	GetPromisedHashrateMin() uint64
	GetPrivateKey() string
	GetBuyerAddress() string
	SetBuyerAddress(buyer string)
	GetCurrentNonce() uint64
	Save() (ISellerContractModel, error)
	Execute() (ISellerContractModel, error)
	GetCloseOutType() uint
	TryRunningAt(dest string) (ISellerContractModel, error)
	Initialize() (ISellerContractModel, error)
}
