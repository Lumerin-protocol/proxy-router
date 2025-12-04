package contract

import (
	"context"
	"errors"
	"time"

	"github.com/Lumerin-protocol/proxy-router/internal/repositories/contracts"
	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate"
	"github.com/ethereum/go-ethereum/common"
)

type ControllerFuturesBuyer struct {
	*ContractWatcherBuyer
	autoClaimReward bool
	deliveryAt      time.Time
	stopCh          chan struct{}
	store           *contracts.FuturesEthereum
	privKey         string
}

func NewControllerFuturesBuyer(contract *ContractWatcherBuyer, store *contracts.FuturesEthereum, deliveryAt time.Time, privKey string, autoClaimReward bool) *ControllerFuturesBuyer {
	return &ControllerFuturesBuyer{
		ContractWatcherBuyer: contract,
		deliveryAt:           deliveryAt,
		autoClaimReward:      autoClaimReward,
		store:                store,
		privKey:              privKey,
		stopCh:               make(chan struct{}),
	}
}

func (c *ControllerFuturesBuyer) Run(ctx context.Context) error {
	defer func() {
		_ = c.log.Close()
		close(c.stopCh)
	}()

	deliveryAtTicker := time.NewTimer(time.Until(c.deliveryAt))
	defer deliveryAtTicker.Stop()

	for {
		select {
		case <-deliveryAtTicker.C:
			c.log.Infof("delivery time reached, starting fulfillment")
			c.ContractWatcherBuyer.StartFulfilling(ctx)
		case <-ctx.Done():
			return ctx.Err()
		case <-c.ContractWatcherBuyer.Done():
			err := c.ContractWatcherBuyer.Err()
			if err != nil {
				// contract closed, no need to close it again
				if errors.Is(err, ErrContractClosed) || c.ContractWatcherBuyer.BlockchainState() == hashrate.BlockchainStateAvailable {
					c.log.Warnf("buyer contract ended due to closeout")
					return nil
				}

				// underdelivery or destination unreachable, buyer closes the contract
				c.log.Warnf("buyer contract ended with error: %s", err)

				blameSeller := true
				if errors.Is(err, ErrContractDest) {
					blameSeller = false
				} else if errors.Is(err, ErrShareTimeout) {
					blameSeller = true
				} else if errors.Is(err, ErrUnderdelivery) {
					blameSeller = true
				}

				err = c.store.CloseDelivery(ctx, common.HexToHash(c.ID()), blameSeller, c.privKey)
				if err != nil {
					c.log.Errorf("error closing contract: %s", err)
					c.log.Info("sleeping for 10 seconds")

					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(10 * time.Second):
					}

					continue
				}

				c.log.Warnf("buyer contract closed, with type cancel, blame seller %b", blameSeller)
				return nil
			} else {
				// delivery ok, seller will close the contract
				c.log.Infof("buyer contract ended without an error")
				return nil
			}
		}
	}
}

func (c *ControllerFuturesBuyer) Stop(ctx context.Context) error {
	c.ContractWatcherBuyer.StopFulfilling()
	<-c.stopCh
	return nil
}

func (c *ControllerFuturesBuyer) SyncState(ctx context.Context) error {
	return nil
}
