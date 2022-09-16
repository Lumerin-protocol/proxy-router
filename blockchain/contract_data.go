package blockchain

import (
	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/hashrouter/lib"
)

type ContractBlockchainState uint8

func (s ContractBlockchainState) String() string {
	switch s {
	case ContractBlockchainStateAvailable:
		return "available"
	case ContractBlockchainStateRunning:
		return "running"
	default:
		return "unknown"
	}
}

const (
	ContractBlockchainStateAvailable ContractBlockchainState = iota
	ContractBlockchainStateRunning
)

type ContractData struct {
	Addr                   common.Address
	Buyer                  common.Address
	Seller                 common.Address
	State                  ContractBlockchainState // external state of the contract (state from blockchain)
	Price                  int64
	Limit                  int64
	Speed                  int64
	Length                 int64
	StartingBlockTimestamp int64
	Dest                   lib.Dest
}

func NewContractData(addr, buyer, seller common.Address, state uint8, price, limit, speed, length, startingBlockTimestamp int64, dest lib.Dest) ContractData {
	return ContractData{
		addr,
		buyer,
		seller,
		ContractBlockchainState(state),
		price,
		limit,
		speed,
		length,
		startingBlockTimestamp,
		dest,
	}
}
