package resources

import (
	"context"
	"math/big"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate"
)

type GenericContractManager interface {
	Run(ctx context.Context) error
}

type Contract interface {
	Run(ctx context.Context) error

	Role() ContractRole                        // the role in the contract (buyer or seller)
	State() ContractState                      // the state of the contract (pending or running)
	BlockchainState() hashrate.BlockchainState // the state of the contract in blockchain (pending or running)
	ValidationStage() hashrate.ValidationStage // the stage of the contract validation (only buyer)

	ID() string     // ID is the unique identifier of the contract, for smart contract data source this is the smart contract address
	Seller() string // ID of the seller (address of the seller for smart contract data source)
	Buyer() string  // ID of the buyer (address of the buyer for smart contract data source)
	Dest() string   // string representation of the destination of the contract (IP address for hashrate, stream URL for video stream etc)
	Price() *big.Int

	Balance() *big.Int
	IsDeleted() bool
	HasFutureTerms() bool
	Version() uint32

	StartTime() time.Time
	FulfillmentStartTime() time.Time
	EndTime() time.Time
	Duration() time.Duration
	Elapsed() time.Duration

	ResourceType() string                  // resource is the name of the resource that the contract is for (hashrate, video stream etc)
	ResourceEstimates() map[string]float64 // map of resouce quantitative estimates, for example for hashrate this would be map[string]string{"hashrate GH/S": "1000"}
	ResourceEstimatesActual() map[string]float64
}

type ContractState string

const (
	ContractStatePending ContractState = "pending"
	ContractStateRunning ContractState = "running"
)

func (c ContractState) String() string {
	switch c {
	case ContractStatePending:
		return "pending"
	case ContractStateRunning:
		return "running"
	default:
		return "unknown"
	}
}

type ContractRole string

const (
	ContractRoleBuyer  ContractRole = "buyer"
	ContractRoleSeller ContractRole = "seller"
)

func (c ContractRole) String() string {
	switch c {
	case ContractRoleBuyer:
		return "buyer"
	case ContractRoleSeller:
		return "seller"
	default:
		return "unknown"
	}
}

type ResourceType string
