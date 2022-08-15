package miner

import "context"

type MinerModel interface {
	Run() error // shouldn't be available as public method, should be called when new miner announced
	ChangeDest(addr string, authUser string, authPwd string) error
	GetID() string // get miner unique id (host:port for example)
	GetHashRate() int64
}

type MinerScheduler interface {
	Run(context.Context) error
	SetDestSplit(*DestSplit)
	GetID() string // get miner unique id (host:port for example)
	GetHashRate() int64
}
