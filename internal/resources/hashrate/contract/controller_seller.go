package contract

import (
	"context"

	"github.com/Lumerin-protocol/contracts-go/implementation"
	"github.com/ethereum/go-ethereum/event"
	dataaccess "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/contracts"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
)

type ControllerSeller struct {
	store    *dataaccess.HashrateEthereum
	contract *ContractWatcher
	sub      *event.Subscription
	ch       chan interface{}
	privKey  string
}

func NewControllerSeller(contract *ContractWatcher, sub *event.Subscription, ch chan interface{}) *ControllerSeller {
	return &ControllerSeller{
		contract: contract,
		sub:      sub,
		ch:       ch,
	}
}

func (c *ControllerSeller) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-c.ch:
			return c.controller(ctx, event)
		}
	}
}

func (c *ControllerSeller) controller(ctx context.Context, event interface{}) error {
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

func (c *ControllerSeller) handleContractPurchased(ctx context.Context, event *implementation.ImplementationContractPurchased) error {
	if c.contract.GetState() == resources.ContractStateRunning {
		return nil
	}

	data, err := c.store.GetContract(ctx, c.contract.GetID())
	if err != nil {
		return err
	}
	c.contract.SetData(data)
	c.contract.StartFulfilling(ctx)

	go func() {
		<-c.contract.Done()
		err := c.store.CloseContract(ctx, c.contract.GetID(), dataaccess.CloseoutTypeWithoutClaim, c.privKey)
		if err != nil {
			c.contract.log.Errorf("error closing contract: %s", err)
		}
	}()

	return nil
}

func (c *ControllerSeller) handleContractClosed(ctx context.Context, event *implementation.ImplementationContractClosed) error {
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

func (c *ControllerSeller) handleCipherTextUpdated(ctx context.Context, event *implementation.ImplementationCipherTextUpdated) error {
	return nil
}

func (c *ControllerSeller) handlePurchaseInfoUpdated(ctx context.Context, event *implementation.ImplementationPurchaseInfoUpdated) error {
	return nil
}
