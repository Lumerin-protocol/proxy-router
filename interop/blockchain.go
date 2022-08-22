package interop

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"gitlab.com/TitanInd/hashrouter/config"
	"gitlab.com/TitanInd/hashrouter/lumerinlib"
)

type BlockchainClient = ethclient.Client
type BlockchainAccount = accounts.Account
type BlockchainAddress = common.Address
type BlockchainEvent = types.Log
type BlockchainEventSubscription = ethereum.Subscription
type BlockchainEventQuery = ethereum.FilterQuery

var (
	// ErrNoCode is returned by call and transact operations for which the requested
	// recipient contract to operate on does not exist in the state db or does not
	// have any code associated with it (i.e. suicided).
	ErrNoCode = bind.ErrNoCode

	// ErrNoPendingState is raised when attempting to perform a pending state action
	// on a backend that doesn't implement PendingContractCaller.
	ErrNoPendingState = bind.ErrNoPendingState

	// ErrNoCodeAfterDeploy is returned by WaitDeployed if contract creation leaves
	// an empty contract behind.
	ErrNoCodeAfterDeploy = bind.ErrNoCodeAfterDeploy
)

// ContractCaller defines the methods needed to allow operating with a contract on a read
// only basis.
type ContractCaller = bind.ContractCaller

// PendingContractCaller defines methods to perform contract calls on the pending state.
// Call will try to discover this interface when access to the pending state is requested.
// If the backend does not support the pending state, Call returns ErrNoPendingState.
type PendingContractCaller = bind.PendingContractCaller

// ContractTransactor defines the methods needed to allow operating with a contract
// on a write only basis. Besides the transacting method, the remainder are helpers
// used when the user does not provide some needed values, but rather leaves it up
// to the transactor to decide.
type ContractTransactor = bind.ContractTransactor

// ContractFilterer defines the methods needed to access log events using one-off
// queries or continuous event subscriptions.
type ContractFilterer = bind.ContractFilterer

// DeployBackend wraps the operations needed by WaitMined and WaitDeployed.
type DeployBackend = bind.DeployBackend

// ContractBackend defines the methods needed to work with contracts on a read-write basis.
type ContractBackend = bind.ContractBackend

func NewBlockchainClient(configuration *config.Config, contractManagerAccount common.Address) (client *BlockchainClient, err error) {
	client, err = ethclient.Dial(configuration.EthNode.Address)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return client, err
	}

	fmt.Printf("Connected to rpc client at %v\n", configuration.EthNode.Address)

	var balance *big.Int
	balance, err = client.BalanceAt(context.Background(), contractManagerAccount, nil)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return client, err
	}
	fbalance := new(big.Float)
	fbalance.SetString(balance.String())
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))

	fmt.Println("Balance of contract manager Account:", ethValue, "ETH")

	return client, err
}
