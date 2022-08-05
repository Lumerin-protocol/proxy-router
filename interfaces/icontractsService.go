package interfaces

type IContractsService interface {
	//	Data management
	GetDestinations() []string
	CreateDestination(destUrl string)
	SaveContracts([]IContractModel)
	GetContract(id string) (IContractModel, error)
	GetHashrate() int64

	CheckHashRate(contractId string) bool

	//	Event listeners
	OnContractCreated(func(newContract IContractModel))

	//	Event handlers
	HandleContractCreated(contract IContractModel)
	HandleContractPurchased(IContractModel)
	HandleContractUpdated(IContractModel)
	HandleDestinationUpdated(IContractModel)
	HandleContractClosed(IContractModel)

	HandleBuyerContractPurchased(IContractModel)
	HandleBuyerContractUpdated(IContractModel)
	HandleBuyerDestinationUpdated(IContractModel)
	HandleBuyerContractClosed(IContractModel)
}
