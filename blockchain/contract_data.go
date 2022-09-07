package blockchain

import (
	"time"

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
	State                  ContractBlockchainState
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

func (d ContractData) GetContractEndTime() int64 {
	return d.StartingBlockTimestamp + d.Length
}

func (d ContractData) GetContractEndTimeV2() time.Time {
	return time.Unix(d.StartingBlockTimestamp+d.Length, 0)
}
