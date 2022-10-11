package blockchain

import (
	"context"
	"errors"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gitlab.com/TitanInd/hashrouter/lib"
)

func TestSubscribeToContractEventsReconnectOnInit(t *testing.T) {
	ethClientMock := &EthClientMock{
		SubscribeFilterLogsFunc: func(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
			time.Sleep(100 * time.Millisecond)
			return nil, errors.New("kiki")
		},
	}

	log, _ := lib.NewTestLogger()
	ethGateway, err := NewEthereumGateway(ethClientMock, "", "", log, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, sub, err := ethGateway.SubscribeToContractEvents(context.Background(), common.Address{})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)
	sub.Unsubscribe()

	if ethClientMock.SubscribeFilterLogsCalledTimes < 5 {
		t.Fatalf("expected to reconnect")
	}
}

func TestSubscribeToContractEventsReconnectOnRead(t *testing.T) {
	ethClientMock := &EthClientMock{
		SubscribeFilterLogsFunc: func(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
			a := &clientSubscription{
				ErrCh: make(chan error, 2),
			}
			time.Sleep(100 * time.Millisecond)
			a.ErrCh <- errors.New("kiki")
			t.Log("emitted")
			return a, nil
		},
	}

	log, _ := lib.NewTestLogger()
	ethGateway, err := NewEthereumGateway(ethClientMock, "", "", log, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, sub, err := ethGateway.SubscribeToContractEvents(context.Background(), common.Address{})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)
	sub.Unsubscribe()

	if ethClientMock.SubscribeFilterLogsCalledTimes < 5 {
		t.Fatalf("expected to reconnect")
	}
}
