package lib

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

var (
	ErrInvalidPrivateKey = fmt.Errorf("invalid private key")
)

func DecryptString(str string, privateKey string) (string, error) {
	pkECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return "", WrapError(ErrInvalidPrivateKey, err)
	}

	pkECIES := ecies.ImportECDSA(pkECDSA)
	strDecodedBytes, err := hex.DecodeString(str)
	if err != nil {
		return "", err
	}

	strDecryptedBytes, err := pkECIES.Decrypt(strDecodedBytes, nil, nil)
	if err != nil {
		return "", err
	}

	return string(strDecryptedBytes), nil
}

func EncryptString(str string, publicKeyHex string) (string, error) {
	pubKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return "", err
	}

	urlBytes := []byte(str)

	publicKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return "", err
	}

	pk := ecies.ImportECDSAPublic(publicKey)

	// Encrypt using ECIES
	ciphertext, err := ecies.Encrypt(rand.Reader, pk, urlBytes, nil, nil)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(ciphertext), nil
}

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
		return common.Address{}, WrapError(ErrInvalidPrivateKey, err)
	}

	addr, err := PrivKeyToAddr(privKey)
	if err != nil {
		return common.Address{}, WrapError(ErrInvalidPrivateKey, err)
	}
	return addr, nil
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

type PubKey struct {
	X       common.Hash
	YParity bool
}

func (p *PubKey) String() string {
	return fmt.Sprintf("x: %s, yParity: %t", p.X.String(), p.YParity)
}

func (p *PubKey) Compressed() []byte {
	prefix := byte(0x02)
	if p.YParity { // If the least significant bit of y is 1, it's odd
		prefix = 0x03
	}

	// Append the prefix and the x-coordinate
	compressed := append([]byte{prefix}, p.X.Bytes()...)
	return compressed
}

// PrKeyToCompressedPubKey converts a private key to a compressed public key (yParity, x) for use with validator registry
func PrKeyToCompressedPubKey(prkey *ecdsa.PrivateKey) (yParity bool, x common.Hash, err error) {
	pub := prkey.PublicKey

	x.SetBytes(pub.X.Bytes())
	return pub.Y.Bit(0) == 1, x, nil
}
