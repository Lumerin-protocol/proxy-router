package miner

import (
	"gitlab.com/TitanInd/hashrouter/data"
)

func NewMinerCollection() *data.Collection[MinerScheduler] {
	return data.NewCollection[MinerScheduler]()
}
