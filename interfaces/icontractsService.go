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

	//	Event listeners
	OnContractCreated(func(newContract IContractModel))

	//	Event handlers
	HandleContractCreated(contract IContractModel)
	HandleContractPurchased(IContractModel)
	HandleContractUpdated(IContractModel)
	HandleDestinationUpdated(IContractModel)
	HandleContractClosed(IContractModel)

	// HandleBuyerContractPurchased(IContractModel)
	// HandleBuyerContractUpdated(IContractModel)
	// HandleBuyerDestinationUpdated(IContractModel)
	// HandleBuyerContractClosed(IContractModel)
}
