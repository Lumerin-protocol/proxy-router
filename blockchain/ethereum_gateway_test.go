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
			time.Sleep(50 * time.Millisecond)
			return nil, errors.New("kiki")
		},
	}

	ethGateway, err := NewEthereumGateway(ethClientMock, "", "", lib.NewTestLogger(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	_, sub, err := ethGateway.SubscribeToContractEvents(context.Background(), common.Address{})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(250 * time.Millisecond)
	sub.Unsubscribe()

	if ethClientMock.SubscribeFilterLogsCalledTimes < 4 {
		t.Fatalf("expected to reconnect")
	}
}

func TestSubscribeToContractEventsReconnectOnRead(t *testing.T) {
	ethClientMock := &EthClientMock{
		SubscribeFilterLogsFunc: func(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
			a := &clientSubscription{
				ErrCh: make(chan error, 2),
			}
			time.Sleep(50 * time.Millisecond)
			a.ErrCh <- errors.New("kiki")
			t.Log("emitted")
			return a, nil
		},
	}

	ethGateway, err := NewEthereumGateway(ethClientMock, "", "", lib.NewTestLogger(), nil, true)
	if err != nil {
		t.Fatal(err)
	}
	_, sub, err := ethGateway.SubscribeToContractEvents(context.Background(), common.Address{})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(250 * time.Millisecond)
	sub.Unsubscribe()

	if ethClientMock.SubscribeFilterLogsCalledTimes < 4 {
		t.Fatalf("expected to reconnect")
	}
}
