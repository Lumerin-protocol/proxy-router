package multicall

import (
	"context"
	"math/big"

	"github.com/Lumerin-protocol/contracts-go/v3/multicall3"
)

type MulticallBackend interface {
	Aggregate(ctx context.Context, calls []multicall3.Multicall3Call) (blockNumer *big.Int, returnData [][]byte, err error)
}
