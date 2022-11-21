package miner

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol"
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
	ChangeDest(context.Context, interfaces.IDestination) error
	GetCurrentDifficulty() int

	GetWorkerName() string
	GetHashRateGHS() int
	GetHashRate() protocol.Hashrate
	GetConnectedAt() time.Time

	OnSubmit(cb protocol.OnSubmitHandler) protocol.ListenerHandle
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
	ChangeDest(ctx context.Context, dest interfaces.IDestination, ID string) error
	GetCurrentDifficulty() int
	GetWorkerName() string
	GetHashRateGHS() int
	GetHashRate() protocol.Hashrate
	GetUnallocatedHashrateGHS() int // get hashrate which is directed to default pool in GH/s
	GetConnectedAt() time.Time
	GetUptime() time.Duration

	RangeDestConn(f func(key any, value any) bool)
	RangeHistory(f func(item HistoryItem) bool)
	RangeHistoryContractID(contractID string, f func(item HistoryItem) bool)
}
