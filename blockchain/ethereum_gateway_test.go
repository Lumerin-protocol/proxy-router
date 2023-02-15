package blockchain

import (
	"context"
	"errors"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"gitlab.com/TitanInd/hashrouter/lib"
)

func TestSubscribeToContractEventsReconnectOnInit(t *testing.T) {
	ethClientMock := &EthClientMock{
		SubscribeFilterLogsFunc: func(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
			time.Sleep(100 * time.Millisecond)
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

	ethGateway, err := NewEthereumGateway(ethClientMock, "", "", lib.NewTestLogger(), nil, true)
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

func TestDecodingContractDataSignedWithSellerPublicKey(t *testing.T) {
	// On buyer side on purchase you get public key, encrypt destination.
	// On create contract Seller set public key, then buyer get it.
	privateKey := "1d9b043a8a288b4510a2085129eb343a5df740773b75fb52882e0072f7d75341"
	ethGateway, err := NewEthereumGateway(nil, privateKey, "", lib.NewTestLogger(), nil, true)
	if err != nil {
		t.Fatal(err)
	}
	encryptedUrl := "0467fede2841ffe5cbbbb741ffa77d83ccc91b76f33f93c1f8257880f6aa7564e0f81be4d58835db231e44a82dcdeed39424384bb835e2590bcd97b70140f8a819ada142285ad76795885712ee037530146942f1c7d82612e44ecfc6c33f7a1633765a59e6e9337f599df99e14b35cd7949187326eb09b6f4e9647b8042d7c64d8ba8ff62f00e69a0a8324dc5faf4841defc39e0fa"
	data, _ := ethGateway.decryptDestination(encryptedUrl)
	t.Log(data)
	assert.Equal(t, "stratum+tcp://josh:josh@1.1.1.1:1000", data, "Incorrect decoded url")
}

func TestIntegrationDecodingContractDestination(t *testing.T) {
	gethNodeAddress := "wss://goerli.infura.io/ws/v3/4b68229d56fe496e899f07c3d41cb08a"
	privateKey := "1d9b043a8a288b4510a2085129eb343a5df740773b75fb52882e0072f7d75341"
	cloneFactory := "0x9d2313d82FD99e61d3d7F616849127c347004A8A"
	encryptedContractAddress := "0x3dC9bf297ba0C86628aF7fABB1AD3c0E5529d39c"

	client, err := NewEthClient(gethNodeAddress, lib.NewTestLogger())
	if err != nil {
		t.Fatal(err)
	}

	ethGateway, err := NewEthereumGateway(client, privateKey, cloneFactory, lib.NewTestLogger(), nil, true)
	if err != nil {
		t.Fatal(err)
	}

	contract, _ := ethGateway.ReadContract(common.HexToAddress(encryptedContractAddress))
	destObj := contract.(ContractData).Dest
	t.Log(destObj)

	assert.Equal(t, "josh", destObj.Username(), "Incorrect username")
	assert.Equal(t, "josh", destObj.Password(), "Incorrect password")
	assert.Equal(t, "1.1.1.1:1000", destObj.GetHost(), "Incorrect host")
}
