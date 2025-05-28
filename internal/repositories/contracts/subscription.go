package contracts

import (
	"context"
	"errors"
	"time"

	"github.com/Lumerin-protocol/contracts-go/v2/clonefactory"
	"github.com/Lumerin-protocol/contracts-go/v2/implementation"
	"github.com/Lumerin-protocol/proxy-router/internal/interfaces"
	"github.com/Lumerin-protocol/proxy-router/internal/lib"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
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

// WatchContractEvents watches for all events from the contract and converts them to the concrete type, using mapper
func WatchContractEvents(ctx context.Context, client EthereumClient, contractAddr common.Address, mapper EventMapper, log interfaces.ILogger) (*lib.Subscription, error) {
	sink := make(chan interface{})

	return lib.NewSubscription(func(quit <-chan struct{}) error {
		defer close(sink)

		query := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddr},
		}
		in := make(chan types.Log)
		defer close(in)

		var lastErr error

		for attempts := 0; true; attempts++ {
			sub, err := client.SubscribeFilterLogs(ctx, query, in)
			if err != nil {
				lastErr = err

				log.Warnf("subscription error, reconnect in %s: %s", RECONNECT_TIMEOUT, lastErr)

				select {
				case <-quit:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(RECONNECT_TIMEOUT):
				}

				continue
			}
			if attempts > 0 {
				log.Warnf("subscription reconnected due to error: %s", lastErr)
			}
			attempts = 0

			defer sub.Unsubscribe()

		EVENTS_LOOP:
			for {
				select {
				case logEntry := <-in:
					event, err := mapper(logEntry)
					if err != nil {

						if errors.Is(err, ErrUnknownEvent) {
							log.Warnf("unknown event: %s", err)
							continue
						}
						// mapper error, retry won't help
						return err
					}

					select {
					case sink <- event:
					case err := <-sub.Err():
						lastErr = err
						break EVENTS_LOOP
					case <-quit:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				case err := <-sub.Err():
					lastErr = err
					break EVENTS_LOOP
				case <-quit:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			log.Warnf("subscription error, reconnect in %s: %s", RECONNECT_TIMEOUT, lastErr)

			select {
			case <-quit:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(RECONNECT_TIMEOUT):
			}
		}

		return lastErr
	}, sink), nil
}
