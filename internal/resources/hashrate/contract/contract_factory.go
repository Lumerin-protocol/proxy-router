package contract

import (
	"fmt"

	contractmanager "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/contractmanager"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
)

type ContractFactory struct {
	allocator *allocator.Allocator
	log       interfaces.ILogger
}

func NewContractFactory(allocator *allocator.Allocator, log interfaces.ILogger) *ContractFactory {
	return &ContractFactory{
		allocator: allocator,
		log:       log,
	}
}

func (c *ContractFactory) CreateContract(contractData *contractmanager.ContractData) (contractmanager.Contract, error) {
	if contractData.ResourceType != ResourceTypeHashrate {
		panic("unknown resource type " + contractData.ResourceType)
	}
	switch contractData.ContractRole {
	case contractmanager.ContractRoleSeller:
		return NewContractWatcherSeller(contractData, c.allocator, c.log.Named(contractData.ContractID)), nil
	case contractmanager.ContractRoleBuyer:
		return NewContractWatcherBuyer(contractData, c.allocator, c.log.Named(contractData.ContractID)), nil
	default:
		return nil, fmt.Errorf("unknown contract role: %s", contractData.ContractRole)
	}
}

func (c *ContractFactory) GetType() contractmanager.ResourceType {
	return ResourceTypeHashrate
}
