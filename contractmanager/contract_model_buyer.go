package contractmanager

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

// BTCHashrateContract represents the collection of mining resources (collection of miners / parts of the miners) that work to fulfill single contract and monotoring tools of their performance
type BTCBuyerHashrateContract struct {
	*BTCHashrateContract
}

func NewBuyerContract(
	data blockchain.ContractData,
	blockchain interfaces.IBlockchainGateway,
	globalScheduler *GlobalSchedulerService,
	log interfaces.ILogger,
	hr *hashrate.Hashrate,
	isBuyer bool,
	hashrateDiffThreshold float64,
	validationBufferPeriod time.Duration,
	defaultDestination interfaces.IDestination,
) *BTCHashrateContract {

	if hr == nil {
		hr = hashrate.NewHashrate(log)
	}

	contract := &BTCHashrateContract{
		blockchain:             blockchain,
		data:                   data,
		hashrate:               hr,
		log:                    log,
		isBuyer:                isBuyer,
		hashrateDiffThreshold:  hashrateDiffThreshold,
		validationBufferPeriod: validationBufferPeriod,
		globalScheduler:        globalScheduler,
		state:                  convertBlockchainStatusToApplicationStatus(data.State),
		defaultDestination:     defaultDestination,
	}

	return contract
}

func (c *BTCBuyerHashrateContract) FulfillContract(ctx context.Context) error {
	c.state = ContractStatePurchased

	if c.ContractIsExpired() {
		c.log.Warn("contract is expired %s", c.GetID())
		c.Close(ctx)
		return fmt.Errorf("contract is expired")
	}
	//race condition here for some contract closeout scenarios
	// initialization cycle waits for hashpower to be available
	// for {
	err := c.StartHashrateAllocation()
	if err != nil {
		return err
	}
	c.stopFullfillment = make(chan struct{}, 10)

	// running cycle checks combination every N seconds
	for {
		c.log.Debugf("Checking if contract is ready for allocation: %v", c.GetID())

		if c.ContractIsExpired() {
			c.log.Info("contract time ended, or state is closed, closing...", c.GetID())
			return fmt.Errorf("contract is expired")
		}

		c.log.Debugf("Should the contract continue? %v", c.ShouldContractContinue())

		if c.ShouldContractContinue() {

			// TODO hashrate monitoring
			c.log.Infof("contract (%s) is running for %.0f seconds", c.GetID(), time.Since(*c.GetStartTime()).Seconds())

			minerIDs, err := c.globalScheduler.CheckHashrate(ctx, c.minerIDs, c.GetHashrateGHS(), c.GetDest(), c.GetID(), c.hashrateDiffThreshold)
			if err != nil {
				// cancel
				c.log.Errorf("cannot update combination %s", err)
			} else {
				c.minerIDs = minerIDs
			}

			select {
			case <-ctx.Done():
				c.log.Errorf("contract context done while waiting for running contract to finish: %v", ctx.Err().Error())
				return ctx.Err()
			case <-c.stopFullfillment:
				c.log.Infof("Done fullfilling contract %v", c.GetID())
				return nil
			case <-time.After(30 * time.Second):
				continue
			}
		}

		c.log.Debugf("Discontinuing Contract %v", c.GetID())

		return nil
	}
}

func (c *BTCBuyerHashrateContract) StartHashrateAllocation() error {
	c.state = ContractStateRunning

	minerList, err := c.globalScheduler.Allocate(c.GetID(), c.GetHashrateGHS(), c.data.Dest)
	if err != nil {
		return err
	}

	c.minerIDs = minerList.IDs()

	c.log.Infof("fulfilling contract %s; expires at %v", c.GetID(), c.GetEndTime())

	return nil
}
