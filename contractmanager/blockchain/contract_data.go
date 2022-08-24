package blockchain

import (
	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/hashrouter/lib"
)

type ContractData struct {
	State                  uint8
	Price                  int64
	Limit                  int64
	Speed                  int64
	Length                 int64
	StartingBlockTimestamp int64
	Buyer                  common.Address
	Seller                 common.Address
	Dest                   lib.Dest
}

func NewContractData(state uint8, price, limit, speed, length, startingBlockTimestamp int64, buyer, seller common.Address, dest lib.Dest) ContractData {
	return ContractData{
		state,
		price,
		limit,
		speed,
		length,
		startingBlockTimestamp,
		buyer,
		seller,
		dest,
	}
}
