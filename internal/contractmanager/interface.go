package contractmanager

import (
	"context"
	"time"
)

type GenericContractManager interface {
	Run(ctx context.Context) error
}

type Contract interface {
	Run(ctx context.Context) error
	GetRole() ContractRole   // the role in the contract (buyer or seller)
	GetState() ContractState // the state of the contract (pending or running)
	GetID() string           // ID is the unique identifier of the contract, for smart contract data source this is the smart contract address
	GetBuyer() string        // ID of the buyer (address of the buyer for smart contract data source)
	GetSeller() string       // ID of the seller (address of the seller for smart contract data source)
	GetDest() string         // string representation of the destination of the contract (IP address for hashrate, stream URL for video stream etc)

	GetStartedAt() *time.Time
	GetFulfillmentStartedAt() *time.Time
	GetEndTime() *time.Time
	GetDuration() time.Duration

	GetResourceType() string                  // resource is the name of the resource that the contract is for (hashrate, video stream etc)
	GetResourceEstimates() map[string]float64 // map of resouce quantitative estimates, for example for hashrate this would be map[string]string{"hashrate GH/S": "1000"}
	GetResourceEstimatesActual() map[string]float64
}

type ContractState string

const (
	ContractStatePending ContractState = "pending"
	ContractStateRunning ContractState = "running"
)

type ContractRole string

const (
	ContractRoleBuyer  ContractRole = "buyer"
	ContractRoleSeller ContractRole = "seller"
)

type ResourceType string
