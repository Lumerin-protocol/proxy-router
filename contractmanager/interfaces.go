package contractmanager

import (
	"context"
	"time"
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
}
