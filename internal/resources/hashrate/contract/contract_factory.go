package contract

import (
	"fmt"
	"net/url"
	"time"

	"github.com/Lumerin-protocol/proxy-router/internal/interfaces"
	"github.com/Lumerin-protocol/proxy-router/internal/lib"
	"github.com/Lumerin-protocol/proxy-router/internal/repositories/contracts"
	"github.com/Lumerin-protocol/proxy-router/internal/resources"
	hashrateContract "github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate"
	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate/allocator"
	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate/hashrate"
	"github.com/ethereum/go-ethereum/common"
)

type ContractFactory struct {
	// config
	privateKey               string // private key of the user
	cycleDuration            time.Duration
	shareTimeout             time.Duration
	hrErrorThreshold         float64
	hashrateCounterNameBuyer string
	validatorFlatness        time.Duration
	validatorStartTime       time.Time
	defaultDest              *url.URL

	// state
	address common.Address // derived from private key

	// deps
	store           *contracts.HashrateEthereum
	allocator       *allocator.Allocator
	globalHashrate  *hashrate.GlobalHashrate
	hashrateFactory func() *hashrate.Hashrate
	logFactory      func(contractID string) (interfaces.ILogger, error)
}

func NewContractFactory(
	allocator *allocator.Allocator,
	hashrateFactory func() *hashrate.Hashrate,
	globalHashrate *hashrate.GlobalHashrate,
	store *contracts.HashrateEthereum,
	logFactory func(contractID string) (interfaces.ILogger, error),

	privateKey string,
	cycleDuration time.Duration,
	shareTimeout time.Duration,
	hrErrorThreshold float64,
	hashrateCounterNameBuyer string,
	validatorFlatness time.Duration,
	validatorStartTime time.Time,
	defaultDest *url.URL,
) (*ContractFactory, error) {
	address, err := lib.PrivKeyStringToAddr(privateKey)
	if err != nil {
		return nil, err
	}

	return &ContractFactory{
		allocator:       allocator,
		hashrateFactory: hashrateFactory,
		globalHashrate:  globalHashrate,
		store:           store,
		logFactory:      logFactory,

		address: address,

		privateKey:               privateKey,
		cycleDuration:            cycleDuration,
		shareTimeout:             shareTimeout,
		hrErrorThreshold:         hrErrorThreshold,
		hashrateCounterNameBuyer: hashrateCounterNameBuyer,
		validatorFlatness:        validatorFlatness,
		validatorStartTime:       validatorStartTime,
		defaultDest:              defaultDest,
	}, nil
}

func (c *ContractFactory) CreateContract(contractData *hashrateContract.EncryptedTerms) (resources.Contract, error) {
	log, err := c.logFactory(contractData.ID())
	if err != nil {
		return nil, err
	}

	logNamed := log.Named("CTR").With("CtrAddr", lib.AddrShort(contractData.ID()))

	if contractData.Seller() == c.address.String() {
		terms := &hashrateContract.Terms{
			BaseTerms:      *contractData.Copy(),
			DestinationURL: nil,
			ValidatorURL:   nil,
		}

		watcher := NewContractWatcherSellerV2(terms, c.cycleDuration, c.hashrateFactory, c.allocator, logNamed)
		return NewControllerSeller(watcher, c.store, c.privateKey), nil
	}

	if contractData.Buyer() == c.address.String() || contractData.Validator() == c.address.String() {
		var role resources.ContractRole
		if contractData.Buyer() == c.address.String() {
			role = resources.ContractRoleBuyer
		} else {
			role = resources.ContractRoleValidator
		}

		var (
			destUrl *url.URL
			destErr error
		)
		if contractData.DestEncrypted != "" {
			destUrl, destErr = c.getDestURL(contractData.DestEncrypted)
		}

		terms := &hashrateContract.Terms{
			BaseTerms:      *contractData.Copy(),
			DestinationURL: destUrl,
			ValidatorURL:   nil,
		}
		watcher := NewContractWatcherBuyer(
			terms,
			c.hashrateFactory,
			c.allocator,
			c.globalHashrate,
			logNamed,

			c.cycleDuration,
			c.shareTimeout,
			c.hrErrorThreshold,
			c.hashrateCounterNameBuyer,
			c.validatorFlatness,
			c.validatorStartTime,
			role,
			c.defaultDest,
		)

		if destErr != nil {
			watcher.contractErr.Store(destErr)
		}
		return NewControllerBuyer(watcher, c.store, c.privateKey, false), nil
	}
	return nil, fmt.Errorf("invalid terms %+v", contractData)
}

func (c *ContractFactory) getDestURL(destEncrypted string) (*url.URL, error) {
	if destEncrypted == "" {
		return nil, nil
	}

	dest, err := lib.DecryptString(destEncrypted, c.privateKey)
	if err != nil {
		return nil, err
	}

	return url.Parse(dest)
}

func (c *ContractFactory) GetType() resources.ResourceType {
	return ResourceTypeHashrate
}
