package blockchain

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type EthClientMock struct {
	SubscribeFilterLogsFunc        func(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	SubscribeFilterLogsCalledTimes int
}

func (c *EthClientMock) ChainID(ctx context.Context) (*big.Int, error) {
	return nil, nil
}
func (c *EthClientMock) BalanceAt(ctx context.Context, addr common.Address, blockNumber *big.Int) (*big.Int, error) {
	return nil, nil
}
func (c *EthClientMock) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return nil, nil
}
func (c *EthClientMock) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return nil, nil
}
func (c *EthClientMock) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return nil, nil
}
func (c *EthClientMock) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	return nil, nil
}
func (c *EthClientMock) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return 0, nil
}
func (c *EthClientMock) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return nil, nil
}
func (c *EthClientMock) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return nil, nil
}
func (c *EthClientMock) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	return 0, nil
}
func (c *EthClientMock) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return nil
}
func (c *EthClientMock) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return nil, nil
}
func (c *EthClientMock) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	if c.SubscribeFilterLogsFunc != nil {
		defer func() {
			c.SubscribeFilterLogsCalledTimes++
		}()
		return c.SubscribeFilterLogsFunc(ctx, query, ch)
	}
	return nil, nil
}
