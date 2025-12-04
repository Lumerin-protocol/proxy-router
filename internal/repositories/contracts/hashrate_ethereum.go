package contracts

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/Lumerin-protocol/contracts-go/v2/clonefactory"
	"github.com/Lumerin-protocol/contracts-go/v2/implementation"
	"github.com/Lumerin-protocol/proxy-router/internal/interfaces"
	"github.com/Lumerin-protocol/proxy-router/internal/lib"
	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate"
	hr "github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate/hashrate"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrNotRunning = fmt.Errorf("the contract is not in the running state")
)

type HashrateEthereum struct {
	// config
	legacyTx         bool // use legacy transaction fee, for local node testing
	clonefactoryAddr common.Address

	// state
	nonce   uint64
	mutex   lib.Mutex
	cfABI   *abi.ABI
	implABI *abi.ABI

	// deps
	cloneFactory *clonefactory.Clonefactory
	client       EthereumClient
	logWatcher   LogWatcher
	log          interfaces.ILogger
}

func NewHashrateEthereum(clonefactoryAddr common.Address, client EthereumClient, logWatcher LogWatcher, log interfaces.ILogger) *HashrateEthereum {
	cf, err := clonefactory.NewClonefactory(clonefactoryAddr, client)
	if err != nil {
		panic("invalid clonefactory ABI")
	}
	cfABI, err := clonefactory.ClonefactoryMetaData.GetAbi()
	if err != nil {
		panic("invalid clonefactory ABI: " + err.Error())
	}
	implABI, err := implementation.ImplementationMetaData.GetAbi()
	if err != nil {
		panic("invalid implementation ABI: " + err.Error())
	}
	return &HashrateEthereum{
		cloneFactory:     cf,
		clonefactoryAddr: clonefactoryAddr,
		client:           client,
		cfABI:            cfABI,
		implABI:          implABI,
		mutex:            lib.NewMutex(),
		logWatcher:       logWatcher,
		log:              log,
	}
}

func (g *HashrateEthereum) SetLegacyTx(legacyTx bool) {
	g.legacyTx = legacyTx
}

func (g *HashrateEthereum) GetPaymentToken(ctx context.Context) (common.Address, error) {
	return g.cloneFactory.PaymentToken(&bind.CallOpts{Context: ctx})
}

func (g *HashrateEthereum) GetFeeToken(ctx context.Context) (common.Address, error) {
	return g.cloneFactory.FeeToken(&bind.CallOpts{Context: ctx})
}

func (g *HashrateEthereum) GetContractsIDs(ctx context.Context) ([]string, error) {
	hashrateContractAddresses, err := g.cloneFactory.GetContractList(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}

	addresses := make([]string, len(hashrateContractAddresses))
	for i, address := range hashrateContractAddresses {
		addresses[i] = address.Hex()
	}

	return addresses, nil
}

func (g *HashrateEthereum) GetContract(ctx context.Context, contractID string) (*hashrate.EncryptedTerms, error) {
	instance, err := implementation.NewImplementation(common.HexToAddress(contractID), g.client)
	if err != nil {
		return nil, err
	}

	data, err := instance.GetPublicVariablesV2(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, lib.WrapError(fmt.Errorf("can't get public variables"), err)
	}

	var (
		startsAt              time.Time
		buyer                 string
		encryptedValidatorURL string

		encryptedDestURL string
		validatorAddr    string
	)

	if data.State == 1 { // running
		startsAt = time.Unix(data.StartingBlockTimestamp.Int64(), 0)
		buyer = data.Buyer.Hex()
		encryptedValidatorURL = data.EncryptedPoolData

		validator, err := instance.Validator(&bind.CallOpts{Context: ctx})
		if err != nil {
			return nil, err
		}
		validatorAddr = validator.Hex()

		destUrl, err := instance.EncrDestURL(&bind.CallOpts{Context: ctx})
		if err != nil {
			return nil, err
		}
		if destUrl != "" {
			encryptedDestURL = destUrl
		}
	}

	terms := &hashrate.EncryptedTerms{
		BaseTerms: *hashrate.NewBaseTerms(
			contractID,
			data.Seller.Hex(),
			buyer,
			validatorAddr,
			startsAt,
			time.Duration(data.Terms.Length.Int64())*time.Second,
			float64(hr.HSToGHS(float64(data.Terms.Speed.Int64()))),
			data.Terms.Price,
			data.Terms.ProfitTarget,
			data.IsDeleted,
			data.Balance,
			data.HasFutureTerms,
			data.Terms.Version,
		),
		ValidatorUrlEncrypted: encryptedValidatorURL,
		DestEncrypted:         encryptedDestURL,
	}

	return terms, nil
}

func (g *HashrateEthereum) EarlyClose(ctx context.Context, contractID string, reason CloseReason, privKey string) error {
	timeout := 2 * time.Minute
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	l := lib.NewMutex()

	err := l.LockCtx(ctx)
	if err != nil {
		err = lib.WrapError(err, fmt.Errorf("close contract lock error %s", timeout))
		g.log.Error(err)
		return err
	}
	defer l.Unlock()

	err = g.earlyClose(ctx, contractID, reason, privKey)
	if err != nil {
		if strings.Contains(err.Error(), "the contract is not in the running state") {
			return ErrNotRunning
		}
		return lib.WrapError(fmt.Errorf("close contract error"), err)
	}

	return err
}

func (g *HashrateEthereum) earlyClose(ctx context.Context, contractID string, reason CloseReason, privKey string) error {
	instance, err := implementation.NewImplementation(common.HexToAddress(contractID), g.client)
	if err != nil {
		g.log.Error(err)
		return err
	}

	transactOpts, err := g.getTransactOpts(ctx, privKey)
	if err != nil {
		g.log.Error(err)
		return err
	}

	tx, err := instance.CloseEarly(transactOpts, uint8(reason))
	if err != nil {
		return err
	}
	g.log.Debugf("closed contract id %s, reason %d nonce %d", contractID, reason, tx.Nonce())

	_, err = bind.WaitMined(ctx, g.client, tx)
	if err != nil {
		g.log.Error(err)
		return err
	}

	return nil
}

func (g *HashrateEthereum) ClaimValidatorReward(ctx context.Context, contractID string, privKey string) error {
	timeout := 2 * time.Minute
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	l := lib.NewMutex()

	err := l.LockCtx(ctx)
	if err != nil {
		err = lib.WrapError(err, fmt.Errorf("claim validator reward lock error %s", timeout))
		g.log.Error(err)
		return err
	}
	defer l.Unlock()

	err = g.claimValidatorReward(ctx, contractID, privKey)
	if err != nil {
		return lib.WrapError(fmt.Errorf("claim validator reward error"), err)
	}

	return err
}

func (g *HashrateEthereum) claimValidatorReward(ctx context.Context, contractID string, privKey string) error {
	instance, err := implementation.NewImplementation(common.HexToAddress(contractID), g.client)
	if err != nil {
		g.log.Error(err)
		return err
	}

	transactOpts, err := g.getTransactOpts(ctx, privKey)
	if err != nil {
		g.log.Error(err)
		return err
	}

	tx, err := instance.ClaimFundsValidator(transactOpts)
	if err != nil {
		return err
	}
	g.log.Debugf("claim reward contract id %s, nonce %d", contractID, tx.Nonce())

	_, err = bind.WaitMined(ctx, g.client, tx)
	if err != nil {
		g.log.Error(err)
		return err
	}

	return nil
}

func (s *HashrateEthereum) CreateCloneFactorySubscription(ctx context.Context, clonefactoryAddr common.Address) (*lib.Subscription, error) {
	return s.logWatcher.Watch(ctx, clonefactoryAddr, CreateEventMapper(clonefactoryEventFactory, s.cfABI), nil)
}

func (s *HashrateEthereum) CreateImplementationSubscription(ctx context.Context, contractAddr common.Address) (*lib.Subscription, error) {
	return s.logWatcher.Watch(ctx, contractAddr, CreateEventMapper(implementationEventFactory, s.implABI), nil)
}

func (g *HashrateEthereum) getTransactOpts(ctx context.Context, privKey string) (*bind.TransactOpts, error) {
	privateKey, err := crypto.HexToECDSA(privKey)
	if err != nil {
		return nil, err
	}

	chainId, err := g.client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	transactOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		return nil, err
	}

	// TODO: deal with likely gasPrice issue so our transaction processes before another pending nonce.
	if g.legacyTx {
		gasPrice, err := g.client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, err
		}
		transactOpts.GasPrice = gasPrice
	}

	transactOpts.Value = big.NewInt(0)
	transactOpts.Context = ctx

	return transactOpts, nil
}
