package miner

type MinerModel interface {
	Run() error // shouldn't be available as public method, should be called when new miner announced
	ChangeDest(addr string, authUser string, authPwd string) error
	GetID() string     // get miner unique id (host:port for example)
	IsAvailable() bool // if miner is available to fulfill a contract
}
