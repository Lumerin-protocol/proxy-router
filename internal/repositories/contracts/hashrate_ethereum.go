package contracts

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/Lumerin-protocol/contracts-go/clonefactory"
	"github.com/Lumerin-protocol/contracts-go/implementation"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
)

type HashrateEthereum struct {
	// config
	legacyTx bool

	// state
	nonce   uint64
	mutex   sync.Mutex
	cfABI   *abi.ABI
	implABI *abi.ABI

	// deps
	cloneFactory *clonefactory.Clonefactory
	client       EthereumClient
	log          interfaces.ILogger
}

func NewHashrateEthereum(clonefactoryAddr common.Address, client EthereumClient, log interfaces.ILogger) *HashrateEthereum {
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
		cloneFactory: cf,
		client:       client,
		cfABI:        cfABI,
		implABI:      implABI,
		log:          log,
	}
}

func (g *HashrateEthereum) SetLegacyTx(legacyTx bool) {
	g.legacyTx = legacyTx
}

func (g *HashrateEthereum) GetContractsIDs(ctx context.Context) ([]string, error) {
	hashrateContractAddresses, err := g.cloneFactory.GetContractList(&bind.CallOpts{Context: ctx})
	if err != nil {
		g.log.Error(err)
		return nil, err
	}

	addresses := make([]string, len(hashrateContractAddresses))
	for i, address := range hashrateContractAddresses {
		addresses[i] = address.Hex()
	}

	return addresses, nil
}

func (g *HashrateEthereum) GetContract(ctx context.Context, contractID string) (*resources.ContractData, error) {
	instance, err := implementation.NewImplementation(common.HexToAddress(contractID), g.client)
	if err != nil {
		g.log.Error(err)
		return nil, err
	}

	data, err := instance.GetPublicVariables(&bind.CallOpts{Context: ctx})
	if err != nil {
		g.log.Error(err)
		return nil, err
	}

	return &resources.ContractData{
		ContractID: contractID,
		Seller:     data.Seller.Hex(),
		Buyer:      data.Buyer.Hex(),
		// EncryptedDest:  data.EncryptedPoolData,
		// StartedAt:      time.Unix(data.StartingBlockTimestamp.Int64(), 0),
		Duration: time.Duration(data.Length.Int64()) * time.Second,
		// HashrateGHS:    hashrate.HSToGHS(float64(data.Speed.Int64())),
		// HasFutureTerms: data.HasFutureTerms,
		// IsDeleted:      data.IsDeleted,
		// State:          data.State,
	}, nil
}

func (g *HashrateEthereum) PurchaseContract(ctx context.Context, contractID string, privKey string) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	opts, err := g.getTransactOpts(ctx, privKey)
	if err != nil {
		g.log.Error(err)
		return err
	}

	_, err = g.cloneFactory.SetPurchaseRentalContract(opts, common.HexToAddress(contractID), "")
	if err != nil {
		g.log.Error(err)
		return err
	}

	watchOpts := &bind.WatchOpts{
		Context: ctx,
	}
	sink := make(chan *clonefactory.ClonefactoryClonefactoryContractPurchased)
	sub, err := g.cloneFactory.WatchClonefactoryContractPurchased(watchOpts, sink, []common.Address{})
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	select {
	case <-sink:
		return nil
	case err := <-sub.Err():
		return err
	case <-ctx.Done():
		return ctx.Err()
	}

}

func (g *HashrateEthereum) CloseContract(ctx context.Context, contractID string, closeoutType CloseoutType, privKey string) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

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

	watchOpts := &bind.WatchOpts{
		Context: ctx,
	}
	sink := make(chan *implementation.ImplementationContractClosed)
	sub, err := instance.WatchContractClosed(watchOpts, sink, []common.Address{})
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	_, err = instance.SetContractCloseOut(transactOpts, big.NewInt(int64(closeoutType)))
	if err != nil {
		g.log.Errorf("cannot close contract %s: %s", contractID, err)
		return err
	}

	select {
	case <-sink:
		return nil
	case err := <-sub.Err():
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *HashrateEthereum) CreateCloneFactorySubscription(ctx context.Context, clonefactoryAddr common.Address) (event.Subscription, <-chan interface{}, error) {
	return WatchContractEvents(ctx, s.client, clonefactoryAddr, CreateEventMapper(clonefactoryEventFactory, s.cfABI))
}

func (s *HashrateEthereum) CreateImplementationSubscription(ctx context.Context, contractAddr common.Address) (event.Subscription, <-chan interface{}, error) {
	return WatchContractEvents(ctx, s.client, contractAddr, CreateEventMapper(implementationEventFactory, s.implABI))
}

func (g *HashrateEthereum) getTransactOpts(ctx context.Context, privKey string) (*bind.TransactOpts, error) {
	privateKey, err := crypto.HexToECDSA(privKey)
	if err != nil {
		g.log.Error(err)
		return nil, err
	}

	chainId, err := g.client.ChainID(ctx)
	if err != nil {
		g.log.Error(err)
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
			g.log.Error(err)
			return nil, err
		}
		transactOpts.GasPrice = gasPrice
	}

	fromAddr, err := lib.PrivKeyToAddr(privateKey)
	if err != nil {
		return nil, err
	}

	nonce, err := g.getNonce(ctx, fromAddr)
	if err != nil {
		return nil, err
	}

	transactOpts.GasLimit = uint64(1_000_000)
	transactOpts.Value = big.NewInt(0)
	transactOpts.Nonce = nonce
	transactOpts.Context = ctx

	return transactOpts, nil
}

func (s *HashrateEthereum) getNonce(ctx context.Context, from common.Address) (*big.Int, error) {
	// TODO: consider assuming that local cached nonce is correct and
	// only retrieve pending nonce from blockchain in case of unlikely error
	s.mutex.Lock()
	defer s.mutex.Unlock()

	nonce := &big.Int{}
	blockchainNonce, err := s.client.PendingNonceAt(ctx, from)
	if err != nil {
		return nonce, err
	}

	if s.nonce > blockchainNonce {
		nonce.SetUint64(s.nonce)
	} else {
		nonce.SetUint64(blockchainNonce)
	}

	s.nonce = nonce.Uint64() + 1

	return nonce, nil
}
