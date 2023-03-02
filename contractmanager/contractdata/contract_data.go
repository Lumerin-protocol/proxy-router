package contractdata

import (
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
)

type ContractBlockchainState uint8

const (
	ContractBlockchainStateAvailable ContractBlockchainState = iota
	ContractBlockchainStateRunning
)

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
	Dest                   interfaces.IDestination
}

func NewContractData(addr, buyer, seller common.Address, state uint8, price, limit, speedHS, lengthSeconds, startingBlockTimeUnix int64, dest interfaces.IDestination) ContractData {
	// making sure workername is set to contract ID to be able to identify contract on buyer side
	dst := enforceWorkerName(dest, addr.String())

	return ContractData{
		addr,
		buyer,
		seller,
		ContractBlockchainState(state),
		price,
		limit,
		speedHS,
		lengthSeconds,
		startingBlockTimeUnix,
		dst,
	}
}

func (c *ContractData) GetID() string {
	return c.GetAddress()
}

func (c *ContractData) GetAddress() string {
	return c.Addr.String()
}

func (c *ContractData) GetBuyerAddress() string {
	return c.Buyer.String()
}

func (c *ContractData) GetSellerAddress() string {
	return c.Seller.String()
}

func (c *ContractData) GetHashrateGHS() int {
	return int(c.Speed / int64(math.Pow10(9)))
}

func (c *ContractData) GetDuration() time.Duration {
	return time.Duration(c.Length) * time.Second
}

func (c *ContractData) GetStartTime() *time.Time {
	startTime := time.Unix(c.StartingBlockTimestamp, 0)
	return &startTime
}

func (c *ContractData) GetEndTime() *time.Time {
	endTime := c.GetStartTime().Add(c.GetDuration())
	return &endTime
}

func (c *ContractData) ContractIsExpired() bool {
	endTime := c.GetEndTime()
	if endTime == nil {
		return false
	}
	return time.Now().After(*endTime)
}

func (c *ContractData) GetDest() interfaces.IDestination {
	return c.Dest
}

func (c *ContractData) GetStateExternal() string {
	return c.State.String()
}

func (d *ContractData) Copy() ContractData {
	return ContractData{
		Addr:                   d.Addr,
		Buyer:                  d.Buyer,
		Seller:                 d.Seller,
		State:                  d.State,
		Price:                  d.Price,
		Limit:                  d.Limit,
		Speed:                  d.Speed,
		Length:                 d.Length,
		StartingBlockTimestamp: d.StartingBlockTimestamp,
		Dest:                   d.Dest,
	}
}

func enforceWorkerName(dest interfaces.IDestination, workername string) interfaces.IDestination {
	if dest.String() == "" {
		return lib.Dest{}
	}
	return lib.NewDest(workername, "", dest.GetHost(), nil)
}
