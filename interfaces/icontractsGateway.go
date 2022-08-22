package interfaces

type IContractsGateway interface {
	GetContract(contractId string) (ISellerContractModel, error)
	SaveContract(ISellerContractModel) (ISellerContractModel, error)
}
