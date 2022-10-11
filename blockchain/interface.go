package blockchain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/hashrouter/interop"
)

type Wallet interface {
	GetAccountAddress() interop.BlockchainAddress
	GetPrivateKey() string
}

type WalletFactory = func(mnemonic string, accountIndex int) (Wallet, error)
type WalletFactory2 interface {
	NewWallet(mnemonic string, accountIndex int) (Wallet, error)
}

type EthereumClient interface {
	bind.ContractBackend
	ChainID(ctx context.Context) (*big.Int, error)
	BalanceAt(ctx context.Context, addr common.Address, blockNumber *big.Int) (*big.Int, error)
}
