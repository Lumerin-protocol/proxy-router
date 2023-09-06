package lib

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func PrivKeyToAddr(privateKey *ecdsa.PrivateKey) (common.Address, error) {
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, fmt.Errorf("error casting public key to ECDSA")
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA), nil
}

func PrivKeyStringToAddr(privateKey string) (common.Address, error) {
	privKey, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return common.Address{}, err
	}

	return PrivKeyToAddr(privKey)
}

func MustPrivKeyToAddr(privateKey *ecdsa.PrivateKey) common.Address {
	addr, err := PrivKeyToAddr(privateKey)
	if err != nil {
		panic(err)
	}
	return addr
}

func MustPrivKeyStringToAddr(privateKey string) common.Address {
	addr, err := PrivKeyStringToAddr(privateKey)
	if err != nil {
		panic(err)
	}
	return addr
}
