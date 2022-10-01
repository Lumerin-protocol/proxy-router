package contractmanager

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/interop"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/miner"
)

func TestListenContractEvents(t *testing.T) {
	t.Skip()

	errChannel := make(chan error)
	eventChannel := make(chan interop.BlockchainEvent)
	eventSubscriptionChannel := testSubscription{
		errChannel: errChannel,
	}

	contractCloseoutCalled := make(chan struct{})
	subscribeToContractEventsCalled := make(chan struct{})

	blockchainGateway := &testBlockchainGateway{
		eventChannel:                    eventChannel,
		eventSubscriptionChannel:        eventSubscriptionChannel,
		contractCloseoutCalled:          contractCloseoutCalled,
		subscribeToContractEventsCalled: subscribeToContractEventsCalled,
	}
	log := &lib.LoggerMock{}

	contract := &BTCHashrateContract{
		log:             log,
		blockchain:      blockchainGateway,
		globalScheduler: NewGlobalScheduler(miner.NewMinerCollection(), log),
	}

	testContext := context.TODO()

	go func(t *testing.T, channel chan interop.BlockchainEvent, subscribeToCloneFactoryEventsCalled chan struct{}, contractCloseoutCalled chan struct{}) {
		<-subscribeToContractEventsCalled

		t.Logf("subscribeToContractEventsCalled")

		topics := []interop.BlockchainHash{blockchain.ContractPurchasedHash}

		t.Logf("sending topic 1: %#+v\n\n", topics[0])
		channel <- interop.BlockchainEvent{
			Topics: topics,
		}

		<-contractCloseoutCalled

		t.Logf("sending topic 2: %#+v\n\n", topics[0])
		channel <- interop.BlockchainEvent{
			Topics: topics,
		}

		<-contractCloseoutCalled

		t.Logf("sending topic3: %#+v\n\n", topics[0])
		channel <- interop.BlockchainEvent{
			Topics: topics,
		}

	}(t, eventChannel, subscribeToContractEventsCalled, contractCloseoutCalled)

	err := contract.listenContractEvents(testContext)

	if err != nil {
		t.Errorf("Expected listenContractEvents to not return an error: %v", err)
	}
}

type testSubscription struct {
	errChannel chan error
}

func (testSubscription) Unsubscribe() {

}

func (s testSubscription) Err() <-chan error {
	return s.errChannel
}

type testBlockchainGateway struct {
	eventChannel                          chan interop.BlockchainEvent
	eventSubscriptionChannel              interop.BlockchainEventSubscription
	subscribeToCloneFactoryEventsCalled chan struct{}
	subscribeToContractEventsCalled       chan struct{}
	contractCloseoutCalled                chan struct{}
}

func (g *testBlockchainGateway) SubscribeToCloneFactoryEvents(ctx context.Context) (chan types.Log, interop.BlockchainEventSubscription, error) {
	g.send(g.subscribeToCloneFactoryEventsCalled)
	return g.eventChannel, g.eventSubscriptionChannel, nil
}

func (g *testBlockchainGateway) send(channel chan<- struct{}) {
	fmt.Println("sending to channel")
	go func() { channel <- struct{}{} }()
}

// SubscribeToContractEvents returns channel with events for particular contract
func (g *testBlockchainGateway) SubscribeToContractEvents(ctx context.Context, contractAddress common.Address) (chan types.Log, ethereum.Subscription, error) {
	defer g.send(g.subscribeToContractEventsCalled)
	return g.eventChannel, g.eventSubscriptionChannel, nil
}

// ReadContract reads contract information encoded in the blockchain
func (g *testBlockchainGateway) ReadContract(contractAddress common.Address) (interface{}, error) {
	return blockchain.ContractData{Length: 5}, nil
}

func (g *testBlockchainGateway) ReadContracts(walletAddr interop.BlockchainAddress, isBuyer bool) ([]interop.BlockchainAddress, error) {
	return nil, nil
}

// SetContractCloseOut closes the contract with specified closeoutType
func (g *testBlockchainGateway) SetContractCloseOut(fromAddress string, contractAddress string, closeoutType int64) error {
	defer g.send(g.contractCloseoutCalled)
	return nil
}

func (g *testBlockchainGateway) GetBalanceWei(ctx context.Context, addr common.Address) (*big.Int, error) {
	return nil, nil
}
