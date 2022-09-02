package blockchain

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

type EthereumWallet struct {
	wallet     *hdwallet.Wallet
	account    accounts.Account
	address    BlockchainAddress
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

	address, err := wallet.Address(account)
	if err != nil {
		return nil, err
	}

	privateKey, err := wallet.PrivateKeyHex(account)
	if err != nil {
		return nil, err
	}

	return &EthereumWallet{wallet: wallet, account: account, address: address, privateKey: privateKey}, nil
}

func (wallet *EthereumWallet) GetAccountAddress() BlockchainAddress {
	return wallet.account.Address
}

func (wallet *EthereumWallet) GetPrivateKey() string {
	return wallet.privateKey
}
