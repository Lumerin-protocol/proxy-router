package blockchain

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

type EthereumWallet struct {
	ethWallet  *hdwallet.Wallet
	account    accounts.Account
	privateKey string
}

func NewEthereumWallet(mnemonic string, accountIndex int) (*EthereumWallet, error) {
	wallet, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		return nil, err
	}

	path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%d", accountIndex))
	account, err := wallet.Derive(path, false)
	if err != nil {
		return nil, err
	}

	privateKey, err := wallet.PrivateKeyHex(account)
	if err != nil {
		return nil, err
	}

	return &EthereumWallet{account: account, privateKey: privateKey, ethWallet: wallet}, nil
}

func (wallet *EthereumWallet) GetAddress() (BlockchainAddress, error) {
	return wallet.ethWallet.Address(wallet.account)
}

func (wallet *EthereumWallet) GetPrivateKey() string {
	return wallet.privateKey
}
