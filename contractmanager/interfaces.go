package contractmanager

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/lib"
)

type IContractModel interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context)
	GetBuyerAddress() string
	GetSellerAddress() string
	GetID() string
	GetAddress() string
	GetHashrateGHS() int
	GetStartTime() *time.Time
	GetDuration() time.Duration
	GetEndTime() *time.Time
	GetState() ContractState
	GetStatusInternal() string
	GetDest() lib.Dest
}
