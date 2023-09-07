package contract

import (
	"context"

	"github.com/Lumerin-protocol/contracts-go/implementation"
	"github.com/ethereum/go-ethereum/event"
	dataaccess "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/contracts"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
)

type ControllerBuyer struct {
	store    *dataaccess.HashrateEthereum
	contract *ContractWatcherBuyer
	sub      *event.Subscription
	ch       chan interface{}
	privKey  string
}

func NewControllerBuyer(contract *ContractWatcherBuyer, sub *event.Subscription, ch chan interface{}) *ControllerBuyer {
	return &ControllerBuyer{
		contract: contract,
		sub:      sub,
		ch:       ch,
	}
}

func (c *ControllerBuyer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-c.ch:
			return c.controller(ctx, event)
		}
	}
}

func (c *ControllerBuyer) controller(ctx context.Context, event interface{}) error {
	switch e := event.(type) {
	case *implementation.ImplementationContractPurchased:
		return c.handleContractPurchased(ctx, e)
	case *implementation.ImplementationContractClosed:
		return c.handleContractClosed(ctx, e)
	case *implementation.ImplementationCipherTextUpdated:
		return c.handleCipherTextUpdated(ctx, e)
	case *implementation.ImplementationPurchaseInfoUpdated:
		return c.handlePurchaseInfoUpdated(ctx, e)
	}
	return nil
}

func (c *ControllerBuyer) handleContractPurchased(ctx context.Context, event *implementation.ImplementationContractPurchased) error {
	return nil
}

func (c *ControllerBuyer) handleContractClosed(ctx context.Context, event *implementation.ImplementationContractClosed) error {
	if c.contract.GetState() == resources.ContractStatePending {
		c.contract.StopFulfilling()
	}

	data, err := c.store.GetContract(ctx, c.contract.GetID())
	if err != nil {
		return err
	}
	c.contract.SetData(data)
	return nil
}

func (c *ControllerBuyer) handleCipherTextUpdated(ctx context.Context, event *implementation.ImplementationCipherTextUpdated) error {
	return nil
}

func (c *ControllerBuyer) handlePurchaseInfoUpdated(ctx context.Context, event *implementation.ImplementationPurchaseInfoUpdated) error {
	return nil
}
