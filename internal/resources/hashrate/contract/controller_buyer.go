package contract

import (
	"context"

	"github.com/Lumerin-protocol/contracts-go/implementation"
	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	dataaccess "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/contracts"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
)

type ControllerBuyer struct {
	*ContractWatcherBuyer
	store   *dataaccess.HashrateEthereum
	tsk     *lib.Task
	privKey string
}

func NewControllerBuyer(contract *ContractWatcherBuyer, store *dataaccess.HashrateEthereum, privKey string) *ControllerBuyer {
	return &ControllerBuyer{
		ContractWatcherBuyer: contract,
		store:                store,
		privKey:              privKey,
	}
}

func (c *ControllerBuyer) Run(ctx context.Context) error {
	sub, err := c.store.CreateImplementationSubscription(ctx, common.HexToAddress(c.GetID()))
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	c.log.Infof("started watching contract as buyer, address %s", c.GetID())

	c.ContractWatcherBuyer.StartFulfilling(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-sub.Events():
			err := c.controller(ctx, event)
			if err != nil {
				return err
			}
		case err := <-sub.Err():
			return err
		case <-c.ContractWatcherBuyer.Done():
			return c.ContractWatcherBuyer.Err()
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
	if c.GetState() == resources.ContractStatePending {
		c.StopFulfilling()
	}

	data, err := c.store.GetContract(ctx, c.GetID())
	if err != nil {
		return err
	}
	c.SetData(data)
	return nil
}

func (c *ControllerBuyer) handleCipherTextUpdated(ctx context.Context, event *implementation.ImplementationCipherTextUpdated) error {
	return nil
}

func (c *ControllerBuyer) handlePurchaseInfoUpdated(ctx context.Context, event *implementation.ImplementationPurchaseInfoUpdated) error {
	return nil
}
