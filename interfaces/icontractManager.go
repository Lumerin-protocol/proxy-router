package interfaces

import (
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type IContractManager interface {
	Start() (err error)
	SetupExistingContracts() (err error)
	ReadContracts() ([]common.Address, error)
	WatchHashrateContract(addr string, hrLogs chan types.Log, hrSub ethereum.Subscription)
}
