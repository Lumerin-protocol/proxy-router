package contract

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
	hashrateContract "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
)

type ContractFactory struct {
	address         common.Address // address of the user
	allocator       *allocator.Allocator
	hashrateFactory func() *hashrate.Hashrate
	cycleDuration   time.Duration
	log             interfaces.ILogger
}

func NewContractFactory(userAddress common.Address, allocator *allocator.Allocator, cycleDuration time.Duration, hashrateFactory func() *hashrate.Hashrate, log interfaces.ILogger) *ContractFactory {
	return &ContractFactory{
		address:         userAddress,
		allocator:       allocator,
		hashrateFactory: hashrateFactory,
		cycleDuration:   cycleDuration,
		log:             log,
	}
}

func (c *ContractFactory) CreateContract(contractData *hashrateContract.Terms) (resources.Contract, error) {
	if contractData.Seller == c.address.String() {
		return NewContractWatcherSeller(contractData, c.cycleDuration, c.hashrateFactory, c.allocator, c.log.Named(lib.AddrShort(contractData.ContractID))), nil
	}
	if contractData.Buyer == c.address.String() {
		return NewContractWatcherBuyer(contractData, c.cycleDuration, c.hashrateFactory, c.allocator, c.log.Named(lib.AddrShort(contractData.ContractID))), nil
	}
	return nil, fmt.Errorf("invalid contract data")
}

func (c *ContractFactory) GetType() resources.ResourceType {
	return ResourceTypeHashrate
}
