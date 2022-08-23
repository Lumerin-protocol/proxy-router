package interfaces

type IContractsService interface {
	//	Data management
	GetDestinations() []string
	CreateDestination(destUrl string)
	SaveContracts([]ISellerContractModel) ([]ISellerContractModel, error)
	GetContract(id string) (ISellerContractModel, error)
	GetHashrate() uint64
	ContractExists(id string) bool
	CheckHashRate(contractId string) bool

	//	Event handlers
	HandleContractCreated(contract ISellerContractModel)
	HandleContractPurchased(dest IDestination, sellerAddress string, buyerAddress string, hashrateGHS int) error
	HandleContractUpdated(price int, time int, hashrate int, lossLimit int)
	HandleDestinationUpdated(dest IDestination)
	HandleContractClosed(model ISellerContractModel)

	// HandleBuyerContractPurchased(IContractModel)
	// HandleBuyerContractUpdated(IContractModel)
	// HandleBuyerDestinationUpdated(IContractModel)
	// HandleBuyerContractClosed(IContractModel)
}
