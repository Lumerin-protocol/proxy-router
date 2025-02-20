package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"os"

	"github.com/Lumerin-protocol/proxy-router/internal/config"
	"github.com/Lumerin-protocol/proxy-router/internal/lib"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Config struct {
	WalletPrivateKey string `env:"WALLET_PRIVATE_KEY"    flag:"wallet-private-key" validate:"required"`
}

func (cfg *Config) SetDefaults() {
}

func logCompressedPublicKey() error {
	fmt.Printf("Compressed public key script\n\n")
	var cfg Config
	err := config.LoadConfig(&cfg, &os.Args)
	if err != nil {
		return err
	}

	prkeyHash := common.HexToHash(cfg.WalletPrivateKey)
	privKey, err := crypto.HexToECDSA(prkeyHash.Hex()[2:])
	if err != nil {
		return err
	}

	yParity, x, err := lib.PrKeyToCompressedPubKey(privKey)
	if err != nil {
		return err
	}

	publicKey, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("invalid public key")
	}

	publicKeyBytes := elliptic.Marshal(publicKey.Curve, publicKey.X, publicKey.Y)

	fmt.Printf("Uncompressed public key:\n%s\n\n", "0x"+common.Bytes2Hex(publicKeyBytes))
	fmt.Printf("Compressed public key:\n")
	fmt.Printf("yParity: %t\nx: %s\n\n", yParity, x.String())
	return nil
}
