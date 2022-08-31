package blockchain

import (
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type BlockchainAddress = common.Address
type BlockchainEventSubscription = ethereum.Subscription
type BlockchainEvent = types.Log

type Wallet interface {
	GetAccountAddress() BlockchainAddress
	GetPrivateKey() string
}

type WalletFactory = func(mnemonic string, accountIndex int) (Wallet, error)
type WalletFactory2 interface {
	NewWallet(mnemonic string, accountIndex int) (Wallet, error)
}
