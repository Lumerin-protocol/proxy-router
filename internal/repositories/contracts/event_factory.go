package contracts

import (
	"github.com/Lumerin-protocol/contracts-go/clonefactory"
	"github.com/Lumerin-protocol/contracts-go/implementation"
	"github.com/ethereum/go-ethereum/core/types"
)

type EventMapper func(types.Log) (interface{}, error)

func implementationEventFactory(name string) interface{} {
	switch name {
	case "contractPurchased":
		return new(implementation.ImplementationContractPurchased)
	case "contractClosed":
		return new(implementation.ImplementationContractClosed)
	case "cipherTextUpdated":
		return new(implementation.ImplementationCipherTextUpdated)
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
	case "purchaseInfoUpdated":
		return new(clonefactory.ClonefactoryPurchaseInfoUpdated)
	case "contractDeleteUpdated":
		return new(clonefactory.ClonefactoryContractDeleteUpdated)
	case "contractClosed":
		return new(clonefactory.ClonefactoryContractClosed)
	default:
		return nil
	}
}
