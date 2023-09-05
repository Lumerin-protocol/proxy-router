package contract

import (
	"fmt"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
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

func (c *ContractFactory) CreateContract(contractData *resources.ContractData) (resources.Contract, error) {
	if contractData.ResourceType != ResourceTypeHashrate {
		panic("unknown resource type " + contractData.ResourceType)
	}
	switch contractData.ContractRole {
	case resources.ContractRoleSeller:
		return NewContractWatcherSeller(contractData, c.cycleDuration, c.hashrateFactory, c.allocator, c.log.Named(lib.AddrShort(contractData.ContractID))), nil
	case resources.ContractRoleBuyer:
		return NewContractWatcherBuyer(contractData, c.allocator, c.log.Named(lib.AddrShort(contractData.ContractID))), nil
	default:
		return nil, fmt.Errorf("unknown contract role: %s", contractData.ContractRole)
	}
}

func (c *ContractFactory) GetType() resources.ResourceType {
	return ResourceTypeHashrate
}
