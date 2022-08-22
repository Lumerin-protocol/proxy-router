package contractmanager

import "gitlab.com/TitanInd/hashrouter/interfaces"

type ContractsGateway struct {
	repository IContractsRepository
}

func (gateway *ContractsGateway) GetContract(id string) (interfaces.ISellerContractModel, error) {
	return gateway.repository.Get(id)

}

func (gateway *ContractsGateway) SaveContract(model interfaces.ISellerContractModel) (interfaces.ISellerContractModel, error) {
	return gateway.repository.Save(model)
}

var _ interfaces.IContractsGateway = (*ContractsGateway)(nil)

func NewContractsGateway(repo IContractsRepository) interfaces.IContractsGateway {
	return &ContractsGateway{
		repository: repo,
	}
}

type IContractsRepository interface {
	interfaces.IRepository[interfaces.ISellerContractModel]
}
