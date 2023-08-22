package contract

import (
	"fmt"
	"time"

	contractmanager "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/contractmanager"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
)

type ContractFactory struct {
	allocator       *allocator.Allocator
	hashrateFactory func() *hashrate.Hashrate
	cycleDuration   time.Duration
	log             interfaces.ILogger
}

func NewContractFactory(allocator *allocator.Allocator, cycleDuration time.Duration, hashrateFactory func() *hashrate.Hashrate, log interfaces.ILogger) *ContractFactory {
	return &ContractFactory{
		allocator:       allocator,
		hashrateFactory: hashrateFactory,
		cycleDuration:   cycleDuration,
		log:             log,
	}
}

func (c *ContractFactory) CreateContract(contractData *contractmanager.ContractData) (contractmanager.Contract, error) {
	if contractData.ResourceType != ResourceTypeHashrate {
		panic("unknown resource type " + contractData.ResourceType)
	}
	switch contractData.ContractRole {
	case contractmanager.ContractRoleSeller:
		return NewContractWatcherSeller(contractData, c.cycleDuration, c.hashrateFactory, c.allocator, c.log.Named(contractData.ContractID)), nil
	case contractmanager.ContractRoleBuyer:
		return NewContractWatcherBuyer(contractData, c.allocator, c.log.Named(contractData.ContractID)), nil
	default:
		return nil, fmt.Errorf("unknown contract role: %s", contractData.ContractRole)
	}
}

func (c *ContractFactory) GetType() contractmanager.ResourceType {
	return ResourceTypeHashrate
}
