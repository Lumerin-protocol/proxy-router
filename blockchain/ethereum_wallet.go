package blockchain

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
	"gitlab.com/TitanInd/hashrouter/interop"
)

type EthereumWallet struct {
	wallet     *hdwallet.Wallet
	account    accounts.Account
	address    interop.BlockchainAddress
	privateKey string
}

func NewEthereumWallet(mnemonic string, accountIndex int, privateKey string, walletAddress string) (*EthereumWallet, error) {

	if privateKey == "" {
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

		return &EthereumWallet{address: address, privateKey: privateKey}, nil
	}

	return &EthereumWallet{address: common.HexToAddress(walletAddress), privateKey: privateKey}, nil
}

func (wallet *EthereumWallet) GetAccountAddress() interop.BlockchainAddress {
	return wallet.address
}

func (wallet *EthereumWallet) GetPrivateKey() string {
	return wallet.privateKey
}
