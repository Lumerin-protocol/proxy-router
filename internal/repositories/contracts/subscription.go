package contracts

import (
	"context"

	"github.com/Lumerin-protocol/contracts-go/clonefactory"
	"github.com/Lumerin-protocol/contracts-go/implementation"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
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
	default:
		return nil
	}
}

// WatchContractEvents watches for all events from the contract and converts them to the concrete type, using mapper
func WatchContractEvents(ctx context.Context, client EthereumClient, contractAddr common.Address, mapper EventMapper) (event.Subscription, <-chan interface{}, error) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddr},
	}
	in := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(ctx, query, in)
	if err != nil {
		return nil, nil, err
	}
	sink := make(chan interface{})

	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		defer close(sink)
		defer close(in)

		for {
			select {
			case log := <-in:
				event, err := mapper(log)
				if err != nil {
					return err
				}

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), sink, nil
}
