package contractmanager

import (
	"time"

	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate"
)

type TermsCommon interface {
	ID() string
	BlockchainState() hashrate.BlockchainState
	Seller() string
	Buyer() string
	Validator() string
	StartTime() time.Time
	Duration() time.Duration
	HashrateGHS() float64
}
