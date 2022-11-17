package lib

import (
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
)

func GetRandomAddr() common.Address {
	return common.BigToAddress(big.NewInt(rand.Int63()))
}
