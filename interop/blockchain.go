package interop

import (
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var AddressStringToSlice = common.HexToAddress

type BlockchainAddress = common.Address
type BlockchainEventSubscription = ethereum.Subscription
type BlockchainEvent = types.Log
type BlockchainHash = common.Hash
