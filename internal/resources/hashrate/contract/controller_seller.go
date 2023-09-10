package contract

import (
	"context"

	"github.com/Lumerin-protocol/contracts-go/implementation"
	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/contracts"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
)

type ControllerSeller struct {
	*ContractWatcherSeller

	store   *contracts.HashrateEthereum
	privKey string
}

func NewControllerSeller(contract *ContractWatcherSeller, store *contracts.HashrateEthereum, privKey string) *ControllerSeller {
	return &ControllerSeller{
		ContractWatcherSeller: contract,
		store:                 store,
		privKey:               privKey,
	}
}

func (c *ControllerSeller) Run(ctx context.Context) error {
	sub, err := c.store.CreateImplementationSubscription(ctx, common.HexToAddress(c.GetID()))
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	c.log.Infof("started watching contract as seller, address %s", c.GetID())

	if c.ShouldBeRunning() {
		c.StartFulfilling(ctx)
	}

	for {
		select {
		case event := <-sub.Events():
			err := c.controller(ctx, event)
			if err != nil {
				c.StopFulfilling()
				return err
			}
		case err := <-sub.Err():
			c.StopFulfilling()
			return err
		case <-c.ContractWatcherSeller.Done():
			err := c.ContractWatcherSeller.Err()
			if err != nil {
				// fulfillment error, buyer will close on underdelivery
				c.log.Warnf("seller contract ended with error: %s", err)
				return nil
			}

			// no error, seller closes the contract after expiration
			c.log.Warnf("seller contract ended without error")
			err = c.store.CloseContract(ctx, c.GetID(), contracts.CloseoutTypeWithoutClaim, c.privKey)
			if err != nil {
				c.log.Errorf("error closing contract: %s", err)
				return nil
			}

			c.log.Warnf("seller contract closed")
			return nil
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
	if c.GetState() == resources.ContractStateRunning {
		return nil
	}

	encryptedTerms, err := c.store.GetContract(ctx, c.GetID())
	if err != nil {
		return err
	}
	terms, err := encryptedTerms.Decrypt(c.privKey)
	if err != nil {
		return err
	}
	c.SetData(terms)

	c.StartFulfilling(ctx)
	return nil
}

func (c *ControllerSeller) handleContractClosed(ctx context.Context, event *implementation.ImplementationContractClosed) error {
	if c.GetState() == resources.ContractStatePending {
		c.StopFulfilling()
	}

	data, err := c.store.GetContract(ctx, c.GetID())
	if err != nil {
		return err
	}
	terms, err := data.Decrypt(c.privKey)
	if err != nil {
		return err
	}
	c.SetData(terms)
	return nil
}

func (c *ControllerSeller) handleCipherTextUpdated(ctx context.Context, event *implementation.ImplementationCipherTextUpdated) error {
	return nil
}

func (c *ControllerSeller) handlePurchaseInfoUpdated(ctx context.Context, event *implementation.ImplementationPurchaseInfoUpdated) error {
	return nil
}
