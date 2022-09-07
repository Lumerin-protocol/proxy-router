package contractmanager

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/lib"
)

type IContractModel interface {
	Run(ctx context.Context) error
	Stop()
	GetBuyerAddress() string
	GetSellerAddress() string
	GetID() string
	GetAddress() string
	GetHashrateGHS() int
	GetStartTime() time.Time
	GetEndTime() time.Time
	GetState() ContractState
	GetDest() lib.Dest
}
