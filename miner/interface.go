package miner

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type Hashrate interface {
	GetHashrate5minAvgGHS() int
	GetHashrate30minAvgGHS() int
	GetHashrate1hAvgGHS() int
}

type MinerModel interface {
	Run(ctx context.Context) error // shouldn't be available as public method, should be called when new miner announced
	ChangeDest(ctx context.Context, dest interfaces.IDestination, onSubmit interfaces.IHashrate) error
	OnFault(func(ctx context.Context))

	GetID() string // get miner unique id (host:port for example)
	GetDest() interfaces.IDestination
	GetCurrentDifficulty() int
	GetWorkerName() string
	GetHashRateGHS() int
	GetHashRate() interfaces.Hashrate
	GetConnectedAt() time.Time
	IsFaulty() bool

	RangeDestConn(f func(key any, value any) bool)
}

type MinerScheduler interface {
	ChangeDest(ctx context.Context, dest interfaces.IDestination, ID string, onSubmit interfaces.IHashrate) error
	Run(context.Context) error

	GetStatus() MinerStatus
	GetID() string // get miner unique id (host:port for example)

	GetCurrentDestSplit() *DestSplit
	GetDestSplit() *DestSplit
	GetUpcomingDestSplit() *DestSplit
	SetDestSplit(*DestSplit)

	GetCurrentDest() interfaces.IDestination // get miner total hashrate in GH/s
	GetCurrentDifficulty() int
	GetWorkerName() string
	GetHashRateGHS() int
	GetHashRate() interfaces.Hashrate
	GetUnallocatedHashrateGHS() int // get hashrate which is directed to default pool in GH/s
	GetConnectedAt() time.Time
	GetUptime() time.Duration

	IsVetting() bool
	IsFaulty() bool

	RangeDestConn(f func(key any, value any) bool)
	RangeHistory(f func(item HistoryItem) bool)
	RangeHistoryContractID(contractID string, f func(item HistoryItem) bool)
}
