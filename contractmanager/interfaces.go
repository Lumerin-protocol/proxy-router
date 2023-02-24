package contractmanager

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type IContractModel interface {
	Run(ctx context.Context) error
	GetBuyerAddress() string
	GetSellerAddress() string
	GetID() string
	GetAddress() string
	GetDeliveredHashrate() interfaces.Hashrate
	GetHashrateGHS() int
	GetStartTime() *time.Time
	GetDuration() time.Duration
	GetEndTime() *time.Time
	GetState() ContractState
	GetStateExternal() string
	GetDest() interfaces.IDestination
	IsValidWallet(walletAddress common.Address) bool
	IsBuyer() bool
}
