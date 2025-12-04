package contract

import (
	"context"
	"time"
)

type ControllerFuturesSeller struct {
	*ContractWatcherSellerV2
	deliveryAt time.Time
	stopCh     chan struct{}
}

func NewControllerFuturesSeller(contract *ContractWatcherSellerV2, deliveryAt time.Time) *ControllerFuturesSeller {
	return &ControllerFuturesSeller{
		ContractWatcherSellerV2: contract,
		stopCh:                  make(chan struct{}),
	}
}

func (c *ControllerFuturesSeller) Run(ctx context.Context) error {
	defer func() {
		_ = c.log.Close()
		close(c.stopCh)
	}()

	c.log.Infof("started watching futures contract as seller, address %s", c.ID())

	deliveryAtTicker := time.NewTimer(time.Until(c.deliveryAt))
	defer deliveryAtTicker.Stop()

	for {
		select {
		case <-deliveryAtTicker.C:
			c.log.Infof("delivery at reached, starting fulfillment")
			err := c.StartFulfilling()
			if err != nil {
				c.log.Errorf("error starting fulfillment: %s", err)
			}
		case <-ctx.Done():
			c.log.Infof("context done, stopping contract watcher")
			if c.IsRunning() {
				c.ContractWatcherSellerV2.StopFulfilling()
				c.log.Infof("waiting for contract watcher to stop")
				<-c.ContractWatcherSellerV2.Done()
				c.log.Infof("contract watcher stopped")
			}
			return ctx.Err()
		case <-c.ContractWatcherSellerV2.Done():
			err := c.ContractWatcherSellerV2.Err()
			if err != nil {
				// fulfillment error, buyer will close on underdelivery
				c.log.Warnf("seller contract ended with error: %s", err)
				c.ContractWatcherSellerV2.Reset()
				return err
			}

			// no error, seller closes the contract after expiration
			c.log.Infof("seller contract ended without error")
			c.ContractWatcherSellerV2.Reset()
			return nil
		}
	}
}

func (c *ControllerFuturesSeller) SyncState(ctx context.Context) error {
	return nil
}

func (c *ControllerFuturesSeller) Stop(ctx context.Context) error {
	c.ContractWatcherSellerV2.StopFulfilling()
	<-c.stopCh
	return nil
}
