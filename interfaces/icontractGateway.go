package interfaces

type IContractGateway interface {
	GetContract(contractId string) IContractModel
	SaveContract(IContractModel)
}
