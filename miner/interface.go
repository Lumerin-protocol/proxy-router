package miner

import (
	"context"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type MinerModel interface {
	Run() error // shouldn't be available as public method, should be called when new miner announced
	ChangeDest(dest interfaces.IDestination) error
	GetID() string // get miner unique id (host:port for example)
	GetHashRate() uint64
}

type MinerScheduler interface {
	Run(context.Context) error
	SetDestSplit(*DestSplit)
	GetID() string                                                   // get miner unique id (host:port for example)
	GetHashRate() uint64                                             // get miner hashrate in H/s
	GetUnallocatedHashrate() uint64                                  // get hashrate which is directed to default pool in H/s
	Allocate(percentage float64, dest interfaces.IDestination) error // allocates available miner resources
}
