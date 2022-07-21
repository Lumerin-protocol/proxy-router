package miner

type Miner interface {
	GetID() string
	ChangePool(addr string, username string, password string) error
}
