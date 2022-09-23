package blockchain

import (
	"context"
	"encoding/hex"
	"math/big"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/ethclient"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/lumerinlib/clonefactory"
	"gitlab.com/TitanInd/hashrouter/lumerinlib/implementation"
)

type EthereumGateway struct {
	client                 *ethclient.Client
	cloneFactory           *clonefactory.Clonefactory
	sellerPrivateKeyString string
	cloneFactoryAddr       common.Address
	log                    interfaces.ILogger
	pendingNonce           PendingNonce
}

type PendingNonce struct {
	pendingNonce uint64
	mutex        sync.Mutex
}

func (n *PendingNonce) Lock() {
	n.mutex.Lock()
}
func (n *PendingNonce) Unlock() {
	n.mutex.Unlock()
}
func (n *PendingNonce) SetNonce(nonce uint64) {
	n.pendingNonce = nonce
}

func (n *PendingNonce) GetNonce() uint64 {
	return n.pendingNonce
}

func NewEthereumGateway(ethClient *ethclient.Client, privateKeyString string, cloneFactoryAddrStr string, log interfaces.ILogger) (*EthereumGateway, error) {
	// TODO: extract it to dependency injection, because we'll going to have only one cloneFactory per project
	cloneFactoryAddr := common.HexToAddress(cloneFactoryAddrStr)
	cloneFactory, err := clonefactory.NewClonefactory(cloneFactoryAddr, ethClient)
	if err != nil {
		return nil, err
	}

	pendingNonce := PendingNonce{mutex: sync.Mutex{}}

	return &EthereumGateway{
		client:                 ethClient,
		sellerPrivateKeyString: privateKeyString,
		cloneFactoryAddr:       common.HexToAddress(cloneFactoryAddrStr),
		cloneFactory:           cloneFactory,
		log:                    log,
		pendingNonce:           pendingNonce,
	}, nil

}

// SubscribeToContractCreatedEvent returns channel with events like new contract creation
func (g *EthereumGateway) SubscribeToContractCreatedEvent(ctx context.Context) (chan types.Log, interop.BlockchainEventSubscription, error) {
	return g.SubscribeToContractEvents(ctx, g.cloneFactoryAddr)
}

// SubscribeToContractEvents returns channel with events for particular contract
func (g *EthereumGateway) SubscribeToContractEvents(ctx context.Context, contractAddress common.Address) (chan types.Log, ethereum.Subscription, error) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
	}

	timeoutContext, cancelFunc := context.WithTimeout(ctx, 15*time.Second)

	logs := make(chan types.Log)

	sub, err := g.client.SubscribeFilterLogs(timeoutContext, query, logs)

	if err != nil {
		cancelFunc()
		g.log.Error(err)
		return logs, sub, err
	}

	go func() {
		<-timeoutContext.Done()

		cancelFunc()
	}()

	return logs, sub, nil
}

// ReadContract reads contract information encoded in the blockchain
func (g *EthereumGateway) ReadContract(contractAddress common.Address) (interface{}, error) {
	var contractData ContractData
	instance, err := implementation.NewImplementation(contractAddress, g.client)
	if err != nil {
		g.log.Error(err)
		return contractData, err
	}

	url, err := instance.EncryptedPoolData(nil)
	if err != nil {
		g.log.Error(err)
		return contractData, err
	}

	dest, err := lib.ParseDest(url)
	if err != nil {
		g.log.Error("invalid blockchain contract destination", err)
		return contractData, err
	}

	state, price, limit, speed, length, startingBlockTimestamp, buyer, seller, _, err := instance.GetPublicVariables(&bind.CallOpts{})
	if err != nil {
		g.log.Error(err)
		return contractData, err
	}

	contractData = NewContractData(contractAddress, buyer, seller, state, price.Int64(), limit.Int64(), speed.Int64(), length.Int64(), startingBlockTimestamp.Int64(), dest)

	// TODO: uncomment when encryption is enabled on frontend
	// return g.decryptDest(url)

	return contractData, nil
}

func (g *EthereumGateway) ReadContracts(sellerAccountAddr interop.BlockchainAddress) ([]interop.BlockchainAddress, error) {
	hashrateContractAddresses, err := g.cloneFactory.GetContractList(&bind.CallOpts{})
	if err != nil {
		g.log.Error(err)
		return nil, err
	}

	var sellerContractAddresses []common.Address

	// parse existing hashrate contracts for ones that belong to seller
	for i := range hashrateContractAddresses {
		hashrateContractInstance, err := implementation.NewImplementation(hashrateContractAddresses[i], g.client)
		if err != nil {
			g.log.Error(err)
			return nil, err
		}
		hashrateContractSeller, err := hashrateContractInstance.Seller(nil)
		if err != nil {
			g.log.Error(err)
			return nil, err
		}
		if hashrateContractSeller == sellerAccountAddr {
			sellerContractAddresses = append(sellerContractAddresses, hashrateContractAddresses[i])
		}
	}

	return sellerContractAddresses, nil
}

// SetContractCloseOut closes the contract with specified closeoutType
func (g *EthereumGateway) SetContractCloseOut(fromAddress string, contractAddress string, closeoutType int64) error {
	ctx := context.TODO()

	instance, err := implementation.NewImplementation(common.HexToAddress(contractAddress), g.client)
	if err != nil {
		g.log.Error(err)
		return err
	}

	privateKey, err := crypto.HexToECDSA(g.sellerPrivateKeyString)
	if err != nil {
		g.log.Error(err)
		return err
	}

	chainId, err := g.client.ChainID(ctx)
	if err != nil {
		g.log.Error(err)
		return err
	}
	//TODO: deal with likely gasPrice issue so our transaction processes before another pending nonce.
	// gasPrice, err := g.client.SuggestGasPrice(ctx)
	// if err != nil {
	// 	g.log.Error(err)
	// 	return err
	// }

	g.pendingNonce.Lock()
	nonce, err := g.client.PendingNonceAt(ctx, common.HexToAddress(fromAddress))

	if err != nil {
		g.pendingNonce.Unlock()
		return err
	}
	g.pendingNonce.SetNonce(nonce)

	options, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		return err
	}

	options.GasLimit = uint64(3000000) // in units
	options.Value = big.NewInt(0)      // in wei
	// options.GasPrice = gasPrice
	options.Nonce = big.NewInt(int64(g.pendingNonce.GetNonce()))
	g.log.Debugf("closeout type: %v", closeoutType)

	//TODO: retry if price is too low
	tx, err := instance.SetContractCloseOut(options, big.NewInt(closeoutType))

	g.pendingNonce.Unlock()
	if err != nil {
		g.log.Errorf("cannot close transaction: %s tx: %s fromAddr: %s contractAddr: %s", err, tx, fromAddress, contractAddress)
		return err
	}

	g.log.Infof("contract %s closed, tx: %s", contractAddress, tx.Hash().Hex())
	return nil
}

func (g *EthereumGateway) GetBalanceWei(ctx context.Context, addr common.Address) (*big.Int, error) {
	return g.client.BalanceAt(ctx, addr, nil)
}

// decryptDest decrypts destination uri which is encrypted with private key of the contract creator
func (g *EthereumGateway) decryptDest(encryptedDestUrl string) (string, error) {
	privateKey, err := crypto.HexToECDSA(g.sellerPrivateKeyString)
	if err != nil {
		g.log.Error(err)
		return "", err
	}

	privateKeyECIES := ecies.ImportECDSA(privateKey)
	destUrlBytes, err := hex.DecodeString(encryptedDestUrl)
	if err != nil {
		g.log.Error(err)
		return "", err
	}

	decryptedDestUrlBytes, err := privateKeyECIES.Decrypt(destUrlBytes, nil, nil)
	if err != nil {
		g.log.Error(err)
		return "", err
	}

	return string(decryptedDestUrlBytes), nil
}
