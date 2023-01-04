package lib

import (
	"fmt"
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
)

func GetRandomAddr() common.Address {
	return common.BigToAddress(big.NewInt(rand.Int63()))
}

func AddrShort(addr string) string {
	length := 5
	l := len(addr)
	if l >= length {
		return fmt.Sprintf("0x..%s", addr[l-length:l])
	}
	return addr
}
