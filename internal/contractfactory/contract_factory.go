package contractfactory

import (
	"fmt"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
)

// Loads all resource type handlers. Supposed to be rewritten with dynamic loading
// of resource type handlers using plugin system

type ContractFactoryInterface interface {
	CreateContract(contractData *resources.ContractData) (resources.Contract, error)
	GetType() resources.ResourceType
}

func ContractFactory(factories ...ContractFactoryInterface) func(contractData *resources.ContractData) (resources.Contract, error) {
	return func(contractData *resources.ContractData) (resources.Contract, error) {
		for _, factory := range factories {
			if factory.GetType() == contractData.ResourceType {
				return factory.CreateContract(contractData)
			}
		}
		return nil, fmt.Errorf("unknown resource type %s", contractData.ResourceType)
	}
}
