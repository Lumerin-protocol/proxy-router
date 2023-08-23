package dataaccess

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/Lumerin-protocol/contracts-go/clonefactory"
	"github.com/Lumerin-protocol/contracts-go/implementation"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/contract"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
)

type HashrateEthereum struct {
	client       EthereumClient
	cloneFactory *clonefactory.Clonefactory
	legacyTx     bool
	nonce        uint64
	mutex        sync.Mutex
	log          interfaces.ILogger
}

func NewHashrateEthereum(clonefactoryAddr common.Address, client EthereumClient, log interfaces.ILogger) *HashrateEthereum {
	cf, err := clonefactory.NewClonefactory(clonefactoryAddr, client)
	if err != nil {
		panic("invalid clonefactory ABI")
	}
	return &HashrateEthereum{
		cloneFactory: cf,
		client:       client,
		log:          log,
	}
}

func (g *HashrateEthereum) SetLegacyTx(legacyTx bool) {
	g.legacyTx = legacyTx
}

func (g *HashrateEthereum) GetContractsIDs() ([]string, error) {
	hashrateContractAddresses, err := g.cloneFactory.GetContractList(&bind.CallOpts{})
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

func (g *HashrateEthereum) GetContract(contractID string) (*contract.ContractData, error) {
	instance, err := implementation.NewImplementation(common.HexToAddress(contractID), g.client)
	if err != nil {
		g.log.Error(err)
		return nil, err
	}

	data, err := instance.GetPublicVariables(&bind.CallOpts{})
	if err != nil {
		g.log.Error(err)
		return nil, err
	}

	return &contract.ContractData{
		ContractID:     contractID,
		Seller:         data.Seller.Hex(),
		Buyer:          data.Buyer.Hex(),
		EncryptedDest:  data.EncryptedPoolData,
		StartedAt:      time.Unix(data.StartingBlockTimestamp.Int64(), 0),
		Duration:       time.Duration(data.Length.Int64()) * time.Second,
		HashrateGHS:    hashrate.HSToGHS(float64(data.Speed.Int64())),
		HasFutureTerms: data.HasFutureTerms,
		IsDeleted:      data.IsDeleted,
		State:          data.State,
	}, nil
}

func (g *HashrateEthereum) CloseContract(ctx context.Context, contractID string, closeoutType uint8, privKey string) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	instance, err := implementation.NewImplementation(common.HexToAddress(contractID), g.client)
	if err != nil {
		g.log.Error(err)
		return err
	}

	privateKey, err := crypto.HexToECDSA(privKey)
	if err != nil {
		g.log.Error(err)
		return err
	}

	chainId, err := g.client.ChainID(ctx)
	if err != nil {
		g.log.Error(err)
		return err
	}

	options, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		return err
	}

	// TODO: deal with likely gasPrice issue so our transaction processes before another pending nonce.
	if g.legacyTx {
		gasPrice, err := g.client.SuggestGasPrice(ctx)
		if err != nil {
			g.log.Error(err)
			return err
		}
		options.GasPrice = gasPrice
	}

	fromAddr, err := lib.PrivKeyToAddr(privateKey)
	if err != nil {
		return err
	}

	nonce, err := g.getNonce(ctx, fromAddr)
	if err != nil {
		return err
	}

	options.GasLimit = uint64(1_000_000)
	options.Value = big.NewInt(0)
	options.Nonce = nonce
	options.Context = ctx

	_, err = instance.SetContractCloseOut(options, big.NewInt(int64(closeoutType)))
	if err != nil {
		g.log.Errorf("cannot close contract %s: %s", contractID, err)
		return err
	}

	filterOpts := &bind.FilterOpts{
		Context: ctx,
	}
	iter, err := instance.FilterContractClosed(filterOpts, []common.Address{})
	if err != nil {
		return err
	}
	defer iter.Close()

	if iter.Next() {
		g.log.Infof("contract %s closed", contractID)
		return nil
	}

	return iter.Error()
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
