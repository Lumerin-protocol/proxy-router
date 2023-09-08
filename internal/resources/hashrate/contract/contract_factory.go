package contract

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/contracts"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
	hashrateContract "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
)

type ContractFactory struct {
	// config
	privateKey    string // private key of the user
	cycleDuration time.Duration

	// state
	address common.Address // derived from private key

	// deps
	store           *contracts.HashrateEthereum
	allocator       *allocator.Allocator
	hashrateFactory func() *hashrate.Hashrate
	log             interfaces.ILogger
}

func NewContractFactory(privateKey string, allocator *allocator.Allocator, cycleDuration time.Duration, hashrateFactory func() *hashrate.Hashrate, store *contracts.HashrateEthereum, log interfaces.ILogger) (*ContractFactory, error) {
	address, err := lib.PrivKeyStringToAddr(privateKey)
	if err != nil {
		return nil, err
	}

	return &ContractFactory{
		privateKey:    privateKey,
		cycleDuration: cycleDuration,

		address: address,

		allocator:       allocator,
		hashrateFactory: hashrateFactory,
		store:           store,
		log:             log,
	}, nil
}

func (c *ContractFactory) CreateContract(contractData *hashrateContract.EncryptedTerms) (resources.Contract, error) {
	if contractData.Seller == c.address.String() {
		terms, err := contractData.Decrypt(c.privateKey)
		if err != nil {
			return nil, err
		}
		watcher := NewContractWatcherSeller(terms, c.cycleDuration, c.hashrateFactory, c.allocator, c.log.Named(lib.AddrShort(contractData.ContractID)))
		return NewControllerSeller(watcher, c.store, c.privateKey), nil
	}
	if contractData.Buyer == c.address.String() {
		return NewContractWatcherBuyer(contractData, c.cycleDuration, c.hashrateFactory, c.allocator, c.log.Named(lib.AddrShort(contractData.ContractID))), nil
	}
	return nil, fmt.Errorf("invalid terms %+v", contractData)
}

func (c *ContractFactory) GetType() resources.ResourceType {
	return ResourceTypeHashrate
}
