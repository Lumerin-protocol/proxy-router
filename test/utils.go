package test

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/Lumerin-protocol/contracts-go/clonefactory"
	"github.com/Lumerin-protocol/contracts-go/lumerintoken"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/contracts"
)

var LocalNodeConfig = struct {
	LUMERIN_ADDR      string
	CLONEFACTORY_ADDR string
	ETH_NODE_ADDR     string
	PRIVATE_KEY       string
	PRIVATE_KEY_2     string
	CONTRACT_ID       string
}{
	LUMERIN_ADDR:      "0x5FbDB2315678afecb367f032d93F642f64180aa3",
	CLONEFACTORY_ADDR: "0xa513E6E4b8f2a923D98304ec87F64353C4D5C853",
	ETH_NODE_ADDR:     "ws://localhost:8545",
	PRIVATE_KEY:       "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
	PRIVATE_KEY_2:     "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
	CONTRACT_ID:       "0x9bd03768a7DCc129555dE410FF8E85528A4F88b5",
}

func makeEthClient(t *testing.T) *contracts.EthClient {
	client, err := contracts.DialContext(context.TODO(), LocalNodeConfig.ETH_NODE_ADDR)
	require.NoError(t, err)
	return client
}

func makeEthGatewaySubscription(t *testing.T, client *contracts.EthClient) *contracts.HashrateEthereum {
	watcher := contracts.NewLogWatcherSubscription(client, 5, &lib.LoggerMock{})
	ethGateway := contracts.NewHashrateEthereum(common.HexToAddress(LocalNodeConfig.CLONEFACTORY_ADDR), client, watcher, &lib.LoggerMock{})
	ethGateway.SetLegacyTx(true)
	return ethGateway
}

func makeEthGatewayPolling(t *testing.T, client *contracts.EthClient) *contracts.HashrateEthereum {
	watcher := contracts.NewLogWatcherPolling(client, 5*time.Second, 5, &lib.LoggerMock{})
	ethGateway := contracts.NewHashrateEthereum(common.HexToAddress(LocalNodeConfig.CLONEFACTORY_ADDR), client, watcher, &lib.LoggerMock{})
	ethGateway.SetLegacyTx(true)
	return ethGateway
}

func privateKeyToTransactOpts(ctx context.Context, privKey string, chainID *big.Int) (*bind.TransactOpts, error) {
	privateKey, err := crypto.HexToECDSA(privKey)
	if err != nil {
		return nil, err
	}

	return bind.NewKeyedTransactorWithChainID(privateKey, chainID)
}

func PurchaseContract(ctx context.Context, client contracts.EthereumClient, contractID string, privKey string) error {
	lumerin, err := lumerintoken.NewLumerintoken(common.HexToAddress(LocalNodeConfig.LUMERIN_ADDR), client)
	if err != nil {
		return err
	}

	chainId, err := client.ChainID(ctx)
	if err != nil {
		return err
	}

	opts, err := privateKeyToTransactOpts(ctx, LocalNodeConfig.PRIVATE_KEY_2, chainId)
	if err != nil {
		return err
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return err
	}
	opts.GasPrice = gasPrice

	_, err = lumerin.Approve(opts, common.HexToAddress(LocalNodeConfig.CLONEFACTORY_ADDR), big.NewInt(5*1e8))
	if err != nil {
		return err
	}

	cloneFactory, err := clonefactory.NewClonefactory(common.HexToAddress(LocalNodeConfig.CLONEFACTORY_ADDR), client)
	if err != nil {
		return err
	}

	fee, err := cloneFactory.MarketplaceFee(&bind.CallOpts{})
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(privKey)
	if err != nil {
		return err
	}

	transactOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		return err
	}

	// TODO: deal with likely gasPrice issue so our transaction processes before another pending nonce.
	gasPrice, err = client.SuggestGasPrice(ctx)
	if err != nil {
		return err
	}
	transactOpts.GasPrice = gasPrice
	transactOpts.Context = ctx
	transactOpts.Value = fee

	tx, err := cloneFactory.SetPurchaseRentalContract(transactOpts, common.HexToAddress(contractID), "", 0)
	if err != nil {
		return err
	}

	_, err = bind.WaitMined(ctx, client, tx)
	return err
}

func CreateContract(ctx context.Context, client contracts.EthereumClient, privKey string, hr, price *big.Int, dur time.Duration, dest string) error {
	chainId, err := client.ChainID(ctx)
	if err != nil {
		return err
	}

	cloneFactory, err := clonefactory.NewClonefactory(common.HexToAddress(LocalNodeConfig.CLONEFACTORY_ADDR), client)
	if err != nil {
		return err
	}

	fee, err := cloneFactory.MarketplaceFee(&bind.CallOpts{})
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(privKey)
	if err != nil {
		return err
	}

	transactOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		return err
	}

	// TODO: deal with likely gasPrice issue so our transaction processes before another pending nonce.
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return err
	}
	transactOpts.GasPrice = gasPrice
	transactOpts.Context = ctx
	transactOpts.Value = fee

	tx, err := cloneFactory.SetCreateNewRentalContract(transactOpts, price, new(big.Int), hr, new(big.Int).SetInt64(int64(dur.Seconds())), common.HexToAddress("0x0"), "pubkey")
	if err != nil {
		return err
	}

	_, err = bind.WaitMined(ctx, client, tx)
	//TODO: return contract ID
	return err
}

func transferLMR(t *testing.T, client contracts.EthereumClient, fromPrivateKey string, toAddr common.Address, lmrAmount *big.Int) {
	ctx := context.Background()

	chainID, err := client.ChainID(ctx)
	require.NoError(t, err)

	c2, err := ethclient.Dial(LocalNodeConfig.ETH_NODE_ADDR)
	require.NoError(t, err)

	lumerin, err := lumerintoken.NewLumerintoken(common.HexToAddress(LocalNodeConfig.LUMERIN_ADDR), c2)
	require.NoError(t, err)

	opts, err := privateKeyToTransactOpts(ctx, fromPrivateKey, chainID)
	require.NoError(t, err)

	gasPrice, err := client.SuggestGasPrice(ctx)
	require.NoError(t, err)
	opts.GasPrice = gasPrice

	fmt.Println(opts, toAddr, lmrAmount, fromPrivateKey)
	fmt.Println("lumerin addr", common.HexToAddress(LocalNodeConfig.LUMERIN_ADDR))
	_, err = lumerin.Transfer(opts, toAddr, lmrAmount)
	require.NoError(t, err)
}
