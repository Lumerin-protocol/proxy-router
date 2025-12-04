package contracts

import (
	"time"

	"github.com/Lumerin-protocol/contracts-go/v2/clonefactory"
	"github.com/Lumerin-protocol/contracts-go/v2/implementation"
	"github.com/Lumerin-protocol/contracts-go/v3/futures"
	"github.com/ethereum/go-ethereum/core/types"
)

const RECONNECT_TIMEOUT = 2 * time.Second

type EventMapper func(types.Log) (interface{}, error)

func implementationEventFactory(name string) interface{} {
	switch name {
	case "contractPurchased":
		return new(implementation.ImplementationContractPurchased)
	case "closedEarly":
		return new(implementation.ImplementationClosedEarly)
	case "purchaseInfoUpdated":
		return new(implementation.ImplementationPurchaseInfoUpdated)
	case "destinationUpdated":
		return new(implementation.ImplementationDestinationUpdated)
	case "fundsClaimed":
		return new(implementation.ImplementationFundsClaimed)
	default:
		return nil
	}
}

func clonefactoryEventFactory(name string) interface{} {
	switch name {
	case "contractCreated":
		return new(clonefactory.ClonefactoryContractCreated)
	case "clonefactoryContractPurchased":
		return new(clonefactory.ClonefactoryClonefactoryContractPurchased)
	case "contractDeleteUpdated":
		return new(clonefactory.ClonefactoryContractDeleteUpdated)
	case "purchaseInfoUpdated":
		return new(clonefactory.ClonefactoryPurchaseInfoUpdated)
	default:
		return nil
	}
}

func futuresEventFactory(name string) interface{} {
	switch name {
	case "PositionCreated":
		return new(futures.FuturesPositionCreated)
	case "PositionClosed":
		return new(futures.FuturesPositionClosed)
	case "PositionDeliveryClosed":
		return new(futures.FuturesPositionDeliveryClosed)
	default:
		return nil
	}
}
