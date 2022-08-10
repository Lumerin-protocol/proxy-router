package contractmanager

import "gitlab.com/TitanInd/hashrouter/interfaces"

type ContractsGateway struct {
}

func (gateway *ContractsGateway) GetContract(id string) interfaces.IContractModel {
	panic("ContractsGateway.GetContract unimplemented")
}

func (gateway *ContractsGateway) SaveContract(interfaces.IContractModel) {
	panic("ContractsGateway.SaveContract unimplemented")
}

var _ interfaces.IContractsGateway = (*ContractsGateway)(nil)

func NewContractsGateway() interfaces.IContractsGateway {
	return &ContractsGateway{}
}
