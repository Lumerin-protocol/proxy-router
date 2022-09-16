package blockchain

import "gitlab.com/TitanInd/hashrouter/interop"

type Wallet interface {
	GetAccountAddress() interop.BlockchainAddress
	GetPrivateKey() string
}

type WalletFactory = func(mnemonic string, accountIndex int) (Wallet, error)
type WalletFactory2 interface {
	NewWallet(mnemonic string, accountIndex int) (Wallet, error)
}
