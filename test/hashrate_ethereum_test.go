package test

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/Lumerin-protocol/contracts-go/clonefactory"
	"github.com/Lumerin-protocol/contracts-go/implementation"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/contracts"
)

// These tests require a local ethereum node running on port 8545
// and contracts deployed
// use "yarn hardhat node" or "anvil" to run the node
// and "make node-local-deploy" to deploy contracts

func TestGetContracts(t *testing.T) {
	ethGateway := makeEthGatewaySubscription(t, makeEthClient(t))
	ids, err := ethGateway.GetContractsIDs(context.TODO())
	require.NoError(t, err)
	fmt.Println(ids)
}

func TestGetContract(t *testing.T) {
	ethGateway := makeEthGatewaySubscription(t, makeEthClient(t))
	contract, err := ethGateway.GetContract(context.TODO(), LocalNodeConfig.CONTRACT_ID)
	require.NoError(t, err)
	fmt.Printf("%+v\n", contract)
}

func TestPurchaseContract(t *testing.T) {
	client, err := contracts.DialContext(context.TODO(), LocalNodeConfig.ETH_NODE_ADDR)
	require.NoError(t, err)

	toAddr := lib.MustPrivKeyStringToAddr(LocalNodeConfig.PRIVATE_KEY_2)
	transferLMR(t, client, LocalNodeConfig.PRIVATE_KEY, toAddr, big.NewInt(100*1e8))

	err = PurchaseContract(context.TODO(), client, LocalNodeConfig.CONTRACT_ID, LocalNodeConfig.PRIVATE_KEY_2)
	require.NoError(t, err)
}

func TestCloseContract(t *testing.T) {
	ethGateway := makeEthGatewaySubscription(t, makeEthClient(t))
	err := ethGateway.CloseContract(context.TODO(), LocalNodeConfig.CONTRACT_ID, 0, LocalNodeConfig.PRIVATE_KEY_2)
	require.NoError(t, err)
}

func TestEvents(t *testing.T) {
	ethClient := makeEthClient(t)

	impl := map[string]*contracts.HashrateEthereum{
		"subscription": makeEthGatewaySubscription(t, ethClient),
		"polling":      makeEthGatewayPolling(t, ethClient),
	}

	tst := map[string]func(t *testing.T, ethGateway *contracts.HashrateEthereum){
		"ClonefactoryContractPurchased": testClonefactoryContractPurchasedEvent,
		"ImplementationContractClosed":  testImplementationContractClosed,
		"ClonefactoryContractClosed":    testCloneFactoryContractClosedEvent,
	}

	for name, ethGateway := range impl {
		t.Run(name, func(t *testing.T) {
			for name, test := range tst {
				t.Run(name, func(t *testing.T) {
					test(t, ethGateway)
				})
			}
		})
	}
}

func testClonefactoryContractPurchasedEvent(t *testing.T, ethGateway *contracts.HashrateEthereum) {
	ctx := context.Background()
	sub, err := ethGateway.CreateCloneFactorySubscription(ctx, common.HexToAddress(LocalNodeConfig.CLONEFACTORY_ADDR))
	require.NoError(t, err)
	defer sub.Unsubscribe()

	errCh := make(chan error)

	go func() {
		errCh <- PurchaseContract(ctx, ethGateway.GetClient(), LocalNodeConfig.CONTRACT_ID, LocalNodeConfig.PRIVATE_KEY_2)
	}()

OUTER:
	for {
		select {
		case event := <-sub.Events():
			t.Logf("emitted %T", event)
			purchasedEvent, ok := event.(*clonefactory.ClonefactoryClonefactoryContractPurchased)
			if !ok {
				continue
			}
			require.Equal(t, common.HexToAddress(LocalNodeConfig.CONTRACT_ID), purchasedEvent.Address)
			break OUTER
		case err := <-sub.Err():
			require.NoError(t, err)
		case err := <-errCh:
			require.NoErrorf(t, err, "error while purchasing contract: %s", err)
		}
	}

	err = ethGateway.CloseContract(ctx, LocalNodeConfig.CONTRACT_ID, 0, LocalNodeConfig.PRIVATE_KEY_2)
	require.NoError(t, err)
}

func testCloneFactoryContractClosedEvent(t *testing.T, ethGateway *contracts.HashrateEthereum) {
	ctx := context.Background()

	sub, err := ethGateway.CreateCloneFactorySubscription(ctx, common.HexToAddress(LocalNodeConfig.CLONEFACTORY_ADDR))
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = PurchaseContract(ctx, ethGateway.GetClient(), LocalNodeConfig.CONTRACT_ID, LocalNodeConfig.PRIVATE_KEY_2)
	require.NoError(t, err)

	errCh := make(chan error)
	go func() {
		errCh <- ethGateway.CloseContract(ctx, LocalNodeConfig.CONTRACT_ID, 0, LocalNodeConfig.PRIVATE_KEY_2)
	}()

	for {
		select {
		case event := <-sub.Events():
			t.Logf("emitted %T", event)
			closedEvent, ok := event.(*clonefactory.ClonefactoryContractClosed)
			if !ok {
				continue
			}
			require.Equal(t, LocalNodeConfig.CONTRACT_ID, closedEvent.Address.String())
			return
		case err := <-sub.Err():
			require.NoError(t, err)
		case err := <-errCh:
			require.NoErrorf(t, err, "error while closing contract: %s", err)
		}
	}
}

func testImplementationContractClosed(t *testing.T, ethGateway *contracts.HashrateEthereum) {
	ctx := context.Background()

	err := PurchaseContract(ctx, ethGateway.GetClient(), LocalNodeConfig.CONTRACT_ID, LocalNodeConfig.PRIVATE_KEY_2)
	require.NoError(t, err)

	sub, err := ethGateway.CreateImplementationSubscription(ctx, common.HexToAddress(LocalNodeConfig.CONTRACT_ID))
	require.NoError(t, err)
	defer sub.Unsubscribe()

	errCh := make(chan error)
	go func() {
		errCh <- ethGateway.CloseContract(ctx, LocalNodeConfig.CONTRACT_ID, 0, LocalNodeConfig.PRIVATE_KEY_2)
	}()

	for {
		select {
		case event := <-sub.Events():
			t.Logf("emitted %T", event)
			_, ok := event.(*implementation.ImplementationContractClosed)
			if ok {
				return
			}
			// require.True(t, ok)
			// fix smart contract to emit closer
			// require.Equal(t, lib.MustPrivKeyStringToAddr(LocalNodeConfig.PRIVATE_KEY_2), closedEvent.Buyer)
		case err := <-sub.Err():
			require.NoError(t, err)
		case err := <-errCh:
			require.NoErrorf(t, err, "error while closing contract: %s", err)
		}
	}
}
