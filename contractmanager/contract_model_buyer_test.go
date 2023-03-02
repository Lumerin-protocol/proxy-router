package contractmanager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/contractmanager/contractdata"
	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/lib"
)

func TestCloseoutOnContractEnd(t *testing.T) {
	contractDurationSeconds := 1
	cycleDuration := time.Duration(contractDurationSeconds) * time.Second / 10
	allowance := 2 * cycleDuration

	log := lib.NewTestLogger()

	data := contractdata.GetSampleContractData()
	data.State = contractdata.ContractBlockchainStateRunning
	data.Length = int64(contractDurationSeconds)

	globalHR := NewGlobalHashrateMock()
	globalHR.LoadOrStore(&WorkerHashrateModelMock{
		ID:             data.Dest.Username(),
		HrGHS:          data.GetHashrateGHS(),
		LastSubmitTime: time.Now(),
		TotalWork:      uint64(hashrate.HSToJobSubmitted(hashrate.GHSToHS(data.GetHashrateGHS()) * float64(contractDurationSeconds))),
	})

	ethGateway := blockchain.NewEthereumGatewayMock()

	defaultDest := lib.MustParseDest("stratum+tcp://default:dest@pool.io:1234")
	contract := NewBuyerContract(data, ethGateway, globalHR, log, 0.1, 0, defaultDest, cycleDuration, 7*time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	go func() {
		errCh <- contract.Run(ctx)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Duration(contractDurationSeconds)*time.Second + allowance):
	}

	assert.Equal(t, 1, ethGateway.SetContractCloseOutCalledTimes, "SetContractCloseOut should be called once")
}

func TestContractCloseoutOnEvent(t *testing.T) {
	contractDurationSeconds := 1
	cycleDuration := time.Duration(contractDurationSeconds) * time.Second / 10
	allowance := 3 * cycleDuration
	validationPeriod := 3 * cycleDuration

	log := lib.NewTestLogger()

	data := contractdata.GetSampleContractData()
	data.State = contractdata.ContractBlockchainStateRunning
	data.Length = int64(contractDurationSeconds)

	ethGateway := blockchain.NewEthereumGatewayMock()

	readContractRes := data.Copy()
	readContractRes.State = contractdata.ContractBlockchainStateAvailable
	ethGateway.ReadContractRes = readContractRes

	globalHR := NewGlobalHashrateMock()
	globalHR.LoadOrStore(&WorkerHashrateModelMock{
		ID:             data.Dest.Username(),
		HrGHS:          data.GetHashrateGHS(),
		LastSubmitTime: time.Now(),
		TotalWork:      uint64(hashrate.HSToJobSubmitted(hashrate.GHSToHS(data.GetHashrateGHS()) * float64(contractDurationSeconds))),
	})

	defaultDest := lib.MustParseDest("stratum+tcp://default:dest@pool.io:1234")
	contract := NewBuyerContract(data, ethGateway, globalHR, log, 0.1, validationPeriod, defaultDest, cycleDuration, 7*time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	go func() {
		errCh <- contract.Run(ctx)
	}()

	ethGateway.EmitEvent(types.Log{
		Topics: []common.Hash{blockchain.ContractClosedHash},
	})

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(cycleDuration*5 + allowance):
		assert.Fail(t, "should stop fulfilling buyer contract right away")
	}
	assert.Equal(t, 0, ethGateway.SetContractCloseOutCalledTimes, "SetContractCloseOut should not be called")
}

func TestBuyerEditContractEvent(t *testing.T) {
	contractDurationSeconds := 1
	cycleDuration := time.Duration(contractDurationSeconds) * time.Second / 10
	allowance := 3 * cycleDuration

	log := lib.NewTestLogger()

	data := contractdata.GetSampleContractData()
	data.State = contractdata.ContractBlockchainStateRunning
	data.Length = int64(contractDurationSeconds)

	ethGateway := blockchain.NewEthereumGatewayMock()

	globalHR := NewGlobalHashrateMock()
	globalHR.LoadOrStore(&WorkerHashrateModelMock{
		ID:             data.Dest.Username(),
		HrGHS:          data.GetHashrateGHS(),
		LastSubmitTime: time.Now(),
		TotalWork:      uint64(hashrate.HSToJobSubmitted(hashrate.GHSToHS(data.GetHashrateGHS()) * float64(contractDurationSeconds))),
	})

	defaultDest := lib.MustParseDest("stratum+tcp://default:dest@pool.io:1234")
	contract := NewBuyerContract(data, ethGateway, globalHR, log, 0.1, 5*time.Minute, defaultDest, cycleDuration, 10*time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	go func() {
		errCh <- contract.Run(ctx)
	}()

	// simulating destination change

	readContractRes := data.Copy()
	readContractRes.Dest = lib.MustParseDest("stratum+tcp://updatedworker:@pool.titan.io:3333")
	ethGateway.ReadContractRes = readContractRes

	ethGateway.EmitEvent(types.Log{
		Topics: []common.Hash{blockchain.ContractCipherTextUpdatedHash},
	})

	<-time.After(cycleDuration*2 + allowance)
	assert.True(t, lib.IsEqualDest(contract.GetDest(), readContractRes.Dest), "should update destination on buyer edit event")

	callArgs := globalHR.GetHashRateGHSCallArgs
	lastCallWorkerName := callArgs[len(callArgs)-1][0].(string)
	assert.Equal(t, readContractRes.Dest.Username(), lastCallWorkerName, "should call IsDeliveringAdequateHashrate with updated dest")
}

func TestValidationBufferPeriod(t *testing.T) {
	contractDurationSeconds := 5
	cycleDuration := 100 * time.Millisecond
	allowance := 3 * cycleDuration
	validationBufferPeriod := 4 * cycleDuration

	log := lib.NewTestLogger()

	data := contractdata.GetSampleContractData()
	data.State = contractdata.ContractBlockchainStateRunning
	data.Length = int64(contractDurationSeconds)

	ethGateway := blockchain.NewEthereumGatewayMock()

	contract := NewBuyerContract(
		data,
		ethGateway,
		NewGlobalHashrate(),
		log,

		0.1,
		validationBufferPeriod,
		lib.MustParseDest("stratum+tcp://default:dest@pool.io:1234"),
		cycleDuration,
		10*time.Minute,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	go func() {
		errCh <- contract.Run(ctx)
	}()

	<-time.After(validationBufferPeriod - cycleDuration*2)

	assert.NotEqual(t, ContractStateAvailable, contract.state, "shouldn't close during validation buffer period")
	assert.Equal(t, 0, ethGateway.SetContractCloseOutCalledTimes, "shouldn't close during validation buffer period")

	<-time.After(cycleDuration*2 + allowance)

	assert.Equal(t, ContractStateAvailable, contract.state, "should close right after validation buffer period")
	assert.Equal(t, 1, ethGateway.SetContractCloseOutCalledTimes, "should close right after validation buffer period")
}

func TestBuyerContractIsValid(t *testing.T) {
	buyer := lib.GetRandomAddr()

	data := contractdata.GetSampleContractData()
	data.Buyer = buyer
	data.State = contractdata.ContractBlockchainStateAvailable

	contract := NewBuyerContract(
		data,
		blockchain.NewEthereumGatewayMock(),
		NewGlobalHashrate(),
		lib.NewTestLogger(),

		0.1,
		0,
		lib.MustParseDest("stratum+tcp://default:dest@pool.io:1234"),
		15*time.Minute,
		10*time.Minute,
	)

	contract.data.State = contractdata.ContractBlockchainStateAvailable
	isValid := contract.IsValidWallet(buyer)
	assert.False(t, isValid, "buyer contract shouldn't be valid to run: invalid state available")

	contract.data.State = contractdata.ContractBlockchainStateRunning
	isValid = contract.IsValidWallet(buyer)
	assert.True(t, isValid, "buyer contract should be valid to run")

	contract.data.State = contractdata.ContractBlockchainStateRunning
	contract.data.Buyer = lib.GetRandomAddr()
	isValid = contract.IsValidWallet(buyer)
	assert.False(t, isValid, "buyer contract shouldn't be valid to run: buyer address doesn't match")
}

func TestCloseoutFailure(t *testing.T) {
	contractDurationSeconds := 1
	cycleDuration := time.Duration(contractDurationSeconds) * time.Second / 10
	allowance := 2 * cycleDuration

	log := lib.NewTestLogger()

	data := contractdata.GetSampleContractData()
	data.State = contractdata.ContractBlockchainStateRunning
	data.Length = int64(contractDurationSeconds)

	globalHR := NewGlobalHashrateMock()
	globalHR.LoadOrStore(&WorkerHashrateModelMock{
		ID:             data.Dest.Username(),
		HrGHS:          data.GetHashrateGHS(),
		LastSubmitTime: time.Now(),
		TotalWork:      uint64(hashrate.HSToJobSubmitted(hashrate.GHSToHS(data.GetHashrateGHS()) * float64(contractDurationSeconds))),
	})

	ethGateway := blockchain.NewEthereumGatewayMock()
	ethGateway.SetContractCloseOutErr = errors.New("sample closeout error")

	defaultDest := lib.MustParseDest("stratum+tcp://default:dest@pool.io:1234")
	contract := NewBuyerContract(data, ethGateway, globalHR, log, 0.1, 0, defaultDest, cycleDuration, 7*time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	go func() {
		errCh <- contract.Run(ctx)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Duration(contractDurationSeconds)*time.Second + allowance):
	}

	assert.LessOrEqual(t, 2, ethGateway.SetContractCloseOutCalledTimes, "SetContractCloseOut should be called at least twice")

	ethGateway.SetContractCloseOutErr = nil

	select {
	case err := <-errCh:
		if err != nil {
			assert.Fail(t, "contract should end without an error")
		}
	case <-time.After(cycleDuration * 2):
		assert.Fail(t, "contract should end within two cycles")
	}
}
