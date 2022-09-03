package miner

import (
	"context"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol"
)

type MinerModel interface {
	Run() error    // shouldn't be available as public method, should be called when new miner announced
	GetID() string // get miner unique id (host:port for example)

	GetDest() interfaces.IDestination
	ChangeDest(dest interfaces.IDestination) error
	GetCurrentDifficulty() int
	GetWorkerName() string

	GetHashRateGHS() int
	OnSubmit(cb protocol.OnSubmitHandler) protocol.ListenerHandle

	OnAuthorize(cb protocol.OnAuthorizeHandler) protocol.ListenerHandle
}

type MinerScheduler interface {
	Run(context.Context) error
	GetID() string // get miner unique id (host:port for example)

	GetDestSplit() *DestSplit
	SetDestSplit(*DestSplit)
	GetCurrentDest() interfaces.IDestination // get miner total hashrate in GH/s

	GetCurrentDifficulty() int
	GetWorkerName() string

	GetHashRateGHS() int
	GetUnallocatedHashrateGHS() int // get hashrate which is directed to default pool in GH/s

	Allocate(percentage float64, dest interfaces.IDestination) (*Split, error) // allocates available miner resources
}
