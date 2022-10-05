package interfaces

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gitlab.com/TitanInd/hashrouter/interop"
)

type IBlockchainGateway interface {
	SubscribeToCloneFactoryEvents(ctx context.Context) (chan types.Log, interop.BlockchainEventSubscription, error)

	// SubscribeToContractEvents returns channel with events for particular contract
	SubscribeToContractEvents(ctx context.Context, contractAddress common.Address) (chan types.Log, ethereum.Subscription, error)

	// ReadContract reads contract information encoded in the blockchain
	ReadContract(contractAddress common.Address) (interface{}, error)

	ReadContracts(walletAddr interop.BlockchainAddress, isBuyer bool) ([]interop.BlockchainAddress, error)

	// SetContractCloseOut closes the contract with specified closeoutType
	SetContractCloseOut(fromAddress string, contractAddress string, closeoutType int64) error

	GetBalanceWei(ctx context.Context, addr common.Address) (*big.Int, error)
}
