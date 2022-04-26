package encryption

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"math/big"
)

//encrypts a string using a given public key and returns hex representation of the encoded message
func encryptData(m string, k string) []byte {
	message := []byte(m)
	key := constructPubKey(k)
	pubKeyECIES := ecies.ImportECDSAPublic(&key)
	encryptData, err := ecies.Encrypt(rand.Reader, pubKeyECIES, message, nil, nil)
	if err != nil {
		fmt.Println(err)
	}
	return encryptData
}

//decrypts a ciphertext using a given private key
func decryptData(m []byte, k string) string {
	privKey, _ := crypto.HexToECDSA(k[2:])
	privKeyECIES := ecies.ImportECDSA(privKey)
	decryptdata, _ := privKeyECIES.Decrypt(m, nil, nil)
	return fmt.Sprintf("%s", decryptdata)
}

//turn an ethereum public key into an ECDSA public key
func constructPubKey(k string) ecdsa.PublicKey {
	//split the key into x and y coordinates
	x := k[:len(k)/2]
	y := k[len(k)/2:]
	xBig := new(big.Int)
	yBig := new(big.Int)
	xBig1, _ := xBig.SetString(x, 16)
	yBig1, _ := yBig.SetString(y, 16)
	fmt.Printf("%#v\n", xBig1)
	fmt.Printf("%#v\n", yBig1)
	ecdsaPublicKey := ecdsa.PublicKey{
		Curve: secp256k1.S256(),
		X:     xBig,
		Y:     yBig,
	}

	return ecdsaPublicKey
}
