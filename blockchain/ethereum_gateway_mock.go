package blockchain

import (
	"context"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type EthereumGatewayMock struct {
	ContractEventsChan        chan types.Log
	ReadContractRes           interface{}
	ReadContractErr           error
	SetContractCloseOutErr    error
	SetContractCloseOutCalled bool
}

func NewEthereumGatewayMock() *EthereumGatewayMock {
	return &EthereumGatewayMock{
		ContractEventsChan: make(chan types.Log),
	}
}

func (m *EthereumGatewayMock) GetBalanceWei(ctx context.Context, addr common.Address) (*big.Int, error) {
	return nil, nil
}

func (m *EthereumGatewayMock) ReadContract(addr common.Address) (interface{}, error) {
	return m.ReadContractRes, m.ReadContractErr
}

func (m *EthereumGatewayMock) ReadContracts(addr common.Address, isBuyer bool) ([]common.Address, error) {
	return nil, nil
}

func (m *EthereumGatewayMock) SetContractCloseOut(fromAddress string, contractAddress string, closeoutType int64) error {
	m.SetContractCloseOutCalled = true
	return m.SetContractCloseOutErr
}

func (m *EthereumGatewayMock) SubscribeToCloneFactoryEvents(ctx context.Context) (chan types.Log, ethereum.Subscription, error) {
	return nil, nil, nil
}

func (m *EthereumGatewayMock) SubscribeToContractEvents(ctx context.Context, addr common.Address) (chan types.Log, ethereum.Subscription, error) {
	errCh := make(chan error)
	sub := NewClientSubscription(errCh, func() {})
	fmt.Printf("subscribed....\n")
	return m.ContractEventsChan, sub, nil
}

func (m *EthereumGatewayMock) EmitEvent(event types.Log) {
	m.ContractEventsChan <- event
	fmt.Printf("emitted....\n")
}

var _ interfaces.IBlockchainGateway = new(EthereumGatewayMock)
