package lib

import (
	"math"
	"math/big"
)

func WeiToEth(wei *big.Int) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(math.Pow10(18)))
}
