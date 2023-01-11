package contractmanager

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/lib"
)

func TestContractCloseout(t *testing.T) {
	contractDurationSeconds := 1
	cycleDuration := 1 * time.Second
	allowance := 1 * time.Second

	log := lib.NewTestLogger()
	data := blockchain.ContractData{
		Addr:                   lib.GetRandomAddr(),
		Buyer:                  lib.GetRandomAddr(),
		Seller:                 lib.GetRandomAddr(),
		State:                  blockchain.ContractBlockchainStateAvailable,
		Price:                  10,
		Limit:                  0,
		Speed:                  10,
		Length:                 int64(contractDurationSeconds),
		StartingBlockTimestamp: time.Now().Unix(),
	}

	ethGateway := blockchain.NewEthereumGatewayMock()

	var globalScheduler = &GlobalSchedulerMock{
		IsDeliveringAdequateHashrateRes: true,
	}
	defaultDest := lib.MustParseDest("stratum+tcp://default:dest@pool.io:1234")
	contract := NewBuyerContract(data, ethGateway, globalScheduler, log, hashrate.NewHashrate(), 0.1, 0, defaultDest, cycleDuration)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	go func() {
		// init contract listener
		errCh <- contract.Run(ctx)
	}()

	// setup mock
	data2 := data.Copy()
	data2.State = blockchain.ContractBlockchainStateRunning
	ethGateway.ReadContractRes = data2

	// emit purchase event
	ethGateway.EmitEvent(types.Log{
		Topics: []common.Hash{blockchain.ContractPurchasedHash},
	})

	select {
	case err := <-errCh:
		t.Fatal(err)
	case <-time.After(time.Duration(contractDurationSeconds)*time.Second + allowance):
	}

	if !ethGateway.SetContractCloseOutCalled {
		t.Fatal("SetContractCloseOut not called")
	}
}

func TestContractCloseoutAlreadyStarted(t *testing.T) {
	contractDurationSeconds := 1
	cycleDuration := 1 * time.Second
	allowance := 1 * time.Second

	log := lib.NewTestLogger()
	data := blockchain.ContractData{
		Addr:                   lib.GetRandomAddr(),
		Buyer:                  lib.GetRandomAddr(),
		Seller:                 lib.GetRandomAddr(),
		State:                  blockchain.ContractBlockchainStateRunning,
		Price:                  10,
		Limit:                  0,
		Speed:                  10,
		Length:                 int64(contractDurationSeconds),
		StartingBlockTimestamp: time.Now().Unix(),
	}

	ethGateway := blockchain.NewEthereumGatewayMock()

	var globalScheduler = &GlobalSchedulerMock{
		IsDeliveringAdequateHashrateRes: true,
	}
	defaultDest := lib.MustParseDest("stratum+tcp://default:dest@pool.io:1234")
	contract := NewBuyerContract(data, ethGateway, globalScheduler, log, hashrate.NewHashrate(), 0.1, 0, defaultDest, cycleDuration)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	go func() {
		// init contract listener
		errCh <- contract.Run(ctx)
	}()

	// setup mock
	data2 := data.Copy()
	ethGateway.ReadContractRes = data2

	select {
	case err := <-errCh:
		t.Fatal(err)
	case <-time.After(time.Duration(contractDurationSeconds)*time.Second + allowance):
	}

	if !ethGateway.SetContractCloseOutCalled {
		t.Fatal("SetContractCloseOut not called")
	}
}