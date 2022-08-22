package interfaces

type IContractsService interface {
	//	Data management
	GetDestinations() []string
	CreateDestination(destUrl string)
	SaveContracts([]IContractModel) ([]IContractModel, error)
	GetContract(id string) (IContractModel, error)
	GetHashrate() uint64
	ContractExists(id string) bool
	CheckHashRate(contractId string) bool

	//	Event handlers
	HandleContractCreated(contract IContractModel)
	HandleContractPurchased(dest string, sellerAddress string, buyerAddress string)
	HandleContractUpdated(price int, time int, hashrate int, lossLimit int)
	HandleDestinationUpdated(dest IDestination)
	HandleContractClosed(model IContractModel)

	// HandleBuyerContractPurchased(IContractModel)
	// HandleBuyerContractUpdated(IContractModel)
	// HandleBuyerDestinationUpdated(IContractModel)
	// HandleBuyerContractClosed(IContractModel)
}
