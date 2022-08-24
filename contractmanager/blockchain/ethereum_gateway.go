package blockchain

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/ethclient"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/lumerinlib/implementation"
)

type ContractCreatedHandler = func(addr common.Address)

type EthereumGateway struct {
	client           *ethclient.Client
	privateKeyString string
	log              interfaces.ILogger
}

func NewEthereumGateway(ethClient *ethclient.Client, privateKeyString string) (client *EthereumGateway) {
	return &EthereumGateway{
		client:           ethClient,
		privateKeyString: privateKeyString,
	}
}

// SubscribeToContractEvents returns events for particular contract
func (g *EthereumGateway) SubscribeToContractEvents(contractAddress common.Address) (chan types.Log, ethereum.Subscription, error) {
	ctx := context.TODO()
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
	}

	logs := make(chan types.Log)
	sub, err := g.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		g.log.Error(err)
		return logs, sub, err
	}

	return logs, sub, nil
}

// ReadContract reads contract information encoded in the blockchain
func (g *EthereumGateway) ReadContract(contractAddress common.Address) (ContractData, error) {
	var contractData ContractData
	instance, err := implementation.NewImplementation(contractAddress, g.client)
	if err != nil {
		g.log.Error(err)
		return contractData, err
	}

	g.log.Debugf("Getting Dest url from contract %s", contractAddress)

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

	contractData = NewContractData(state, price.Int64(), limit.Int64(), speed.Int64(), length.Int64(), startingBlockTimestamp.Int64(), buyer, seller, dest)

	// TODO: uncomment when encryption is enabled on frontend
	// return g.decryptDest(url)

	return contractData, nil
}

// ReadContract reads contract information encoded in the blockchain
func (g *EthereumGateway) SetContractCloseOut(fromAddress string, contractAddress string, currentNonce uint64, closeoutType uint) error {
	ctx := context.TODO()
	instance, err := implementation.NewImplementation(common.HexToAddress(contractAddress), g.client)

	if err != nil {
		g.log.Error(err)
		return err
	}

	privateKey, err := crypto.HexToECDSA(g.privateKeyString)
	if err != nil {
		g.log.Error(err)
		return err
	}

	chainId, err := g.client.ChainID(ctx)
	if err != nil {
		g.log.Error(err)
		return err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		g.log.Error(err)
		return err
	}

	gasPrice, err := g.client.SuggestGasPrice(ctx)
	if err != nil {
		g.log.Error(err)
		return err
	}

	currentNonce, err = g.client.PendingNonceAt(ctx, common.HexToAddress(fromAddress))
	if err != nil {
		g.log.Error(err)
		return err
	}

	auth.GasLimit = uint64(3000000) // in units
	auth.Value = big.NewInt(0)      // in wei
	auth.GasPrice = gasPrice
	auth.Nonce = big.NewInt(int64(currentNonce))

	tx, err := instance.SetContractCloseOut(auth, big.NewInt(int64(closeoutType)))
	if err != nil {
		g.log.Error(err)
		return err
	}

	g.log.Infof("tx sent: %s", tx.Hash().Hex())
	g.log.Infof("closing out contract: %s", contractAddress)

	return err
}

func (g *EthereumGateway) decryptDest(encryptedDestUrl string) (string, error) {
	privateKey, err := crypto.HexToECDSA(g.privateKeyString)
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
