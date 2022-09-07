package interfaces

import (
	"context"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ContractManager interface {
	Run(context.Context) (err error)
	SetupExistingContracts() (err error)
	ReadContracts() ([]common.Address, error)
	WatchHashrateContract(addr string, hrLogs chan types.Log, hrSub ethereum.Subscription)
}
