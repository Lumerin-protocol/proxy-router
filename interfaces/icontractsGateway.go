package interfaces

type IContractsGateway interface {
	GetContract(contractId string) (IContractModel, error)
	SaveContract(IContractModel) (IContractModel, error)
}
