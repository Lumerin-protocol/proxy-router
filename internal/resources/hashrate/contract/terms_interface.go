package contract

import (
	"math/big"
	"net/url"
	"time"

	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate"
)

type Terms interface {
	ID() string
	BlockchainState() hashrate.BlockchainState
	Seller() string
	Buyer() string
	Validator() string
	StartTime() time.Time
	EndTime() time.Time
	Duration() time.Duration
	HashrateGHS() float64
	Balance() *big.Int
	Dest() *url.URL
	PoolDest() *url.URL
	Elapsed() time.Duration
	HasFutureTerms() bool
	IsDeleted() bool
	Price() *big.Int
	ProfitTarget() int8
	Version() uint32
	ResetStartTime()
}
