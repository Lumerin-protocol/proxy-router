package encryption

import (
	"C"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

func DecryptData(m string, k string) string {
	privKey, _ := crypto.HexToECDSA(k)
	privKeyECIES := ecies.ImportECDSA(privKey)
	b, _ := hex.DecodeString(m)
	decryptdata, _ := privKeyECIES.Decrypt(b, nil, nil)
	return fmt.Sprintf("%s", decryptdata)
}
