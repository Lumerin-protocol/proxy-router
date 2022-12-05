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
	GetID() string                 // get miner unique id (host:port for example)

	GetDest() interfaces.IDestination
	ChangeDest(ctx context.Context, dest interfaces.IDestination, onSubmit interfaces.IHashrate) error
	GetCurrentDifficulty() int

	GetWorkerName() string
	GetHashRateGHS() int
	GetHashRate() interfaces.Hashrate
	GetConnectedAt() time.Time

	RangeDestConn(f func(key any, value any) bool)
}

type MinerScheduler interface {
	Run(context.Context) error
	GetID() string // get miner unique id (host:port for example)

	IsVetted() bool
	GetStatus() MinerStatus

	GetCurrentDestSplit() *DestSplit
	GetDestSplit() *DestSplit
	GetUpcomingDestSplit() *DestSplit
	SetDestSplit(*DestSplit)

	GetCurrentDest() interfaces.IDestination // get miner total hashrate in GH/s
	ChangeDest(ctx context.Context, dest interfaces.IDestination, ID string, onSubmit interfaces.IHashrate) error
	GetCurrentDifficulty() int
	GetWorkerName() string
	GetHashRateGHS() int
	GetHashRate() interfaces.Hashrate
	GetUnallocatedHashrateGHS() int // get hashrate which is directed to default pool in GH/s
	GetConnectedAt() time.Time
	GetUptime() time.Duration

	RangeDestConn(f func(key any, value any) bool)
	RangeHistory(f func(item HistoryItem) bool)
	RangeHistoryContractID(contractID string, f func(item HistoryItem) bool)
}
