package interfaces

type IContractsGateway interface {
	GetContract(contractId string) IContractModel
	SaveContract(IContractModel)
}
