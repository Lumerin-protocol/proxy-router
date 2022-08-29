package miner

import (
	"context"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol"
)

type MinerModel interface {
	Run() error // shouldn't be available as public method, should be called when new miner announced
	ChangeDest(dest interfaces.IDestination) error
	GetID() string // get miner unique id (host:port for example)
	GetHashRateGHS() int
	OnSubmit(cb protocol.OnSubmitHandler) protocol.ListenerHandle
}

type MinerScheduler interface {
	Run(context.Context) error
	SetDestSplit(*DestSplit)
	GetID() string                                                             // get miner unique id (host:port for example)
	GetHashRateGHS() int                                                       // get miner hashrate in GH/s
	GetUnallocatedHashrateGHS() int                                            // get hashrate which is directed to default pool in GH/s
	Allocate(percentage float64, dest interfaces.IDestination) (*Split, error) // allocates available miner resources
}
