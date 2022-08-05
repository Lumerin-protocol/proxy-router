package blockchain

import (
	"context"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
)

type EthereumGateway struct {
	interfaces.IBlockchainGateway
	client *interop.BlockchainClient
}

func (gateway *EthereumGateway) SubscribeToContractEvents(address string) (chan interop.BlockchainEvent, interop.BlockchainEventSubscription, error) {
	query := ethereum.FilterQuery{
		Addresses: []interop.BlockchainAddress{common.HexToAddress(address)},
	}

	logs := make(chan interop.BlockchainEvent)
	sub, err := gateway.client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		// fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return logs, sub, err
	}

	return logs, sub, err
}

func NewEthereumClientWrapper(ethClient *interop.BlockchainClient) (client interfaces.IBlockchainGateway, err error) {

	return &EthereumGateway{
		client: ethClient,
	}, err
}
