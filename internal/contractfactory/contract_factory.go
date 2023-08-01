package contractfactory

import (
	"fmt"

	cm "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/contractmanager"
)

// Loads all resource type handlers. Supposed to be rewritten with dynamic loading
// of resource type handlers using plugin system

type ContractFactoryInterface interface {
	CreateContract(contractData *cm.ContractData) (cm.Contract, error)
	GetType() cm.ResourceType
}

func ContractFactory(factories ...ContractFactoryInterface) func(contractData *cm.ContractData) (cm.Contract, error) {
	return func(contractData *cm.ContractData) (cm.Contract, error) {
		for _, factory := range factories {
			if factory.GetType() == contractData.ResourceType {
				return factory.CreateContract(contractData)
			}
		}
		return nil, fmt.Errorf("unknown resource type %s", contractData.ResourceType)
	}
}
