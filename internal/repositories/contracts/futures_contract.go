package contracts

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type FuturesContract struct {
	ContractID common.Hash
	Seller     common.Address
	Buyer      common.Address
	DestURL    string
	DeliveryAt time.Time
	Paid       bool
}

func (fc *FuturesContract) ID() string {
	return fc.ContractID.Hex()
}
