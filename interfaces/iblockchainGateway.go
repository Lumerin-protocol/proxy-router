package interfaces

import "gitlab.com/TitanInd/hashrouter/interop"

type IBlockchainGateway interface {
	SubscribeToContractEvents(address string) (chan interop.BlockchainEvent, interop.BlockchainEventSubscription, error)
}
