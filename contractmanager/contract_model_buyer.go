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
	globalScheduler *GlobalSchedulerV2,
	log interfaces.ILogger,
	hr *hashrate.Hashrate,
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
		isBuyer:                true,
		hashrateDiffThreshold:  hashrateDiffThreshold,
		validationBufferPeriod: validationBufferPeriod,
		globalScheduler:        globalScheduler,
		state:                  convertBlockchainStatusToApplicationStatus(data.State),
		defaultDestination:     defaultDestination,
	}

	return contract
}

func (c *BTCHashrateContract) FulfillBuyerContract(ctx context.Context) error {
	c.state = ContractStatePurchased

	if c.ContractIsExpired() {
		c.log.Warn("contract is expired %s", c.GetID())
		return fmt.Errorf("contract is expired")
	}

	// running cycle checks combination every N seconds
	for {
		c.log.Debugf("Checking if contract is ready for allocation: %v", c.GetID())

		if c.ContractIsExpired() {
			c.log.Info("contract time ended, or state is closed, closing...", c.GetID())
			return fmt.Errorf("contract is expired")
		}

		c.log.Debugf("Should the contract continue? %v", c.ShouldContractContinue())

		if !c.ShouldContractContinue() {
			c.log.Debugf("Discontinuing Contract %v", c.GetID())
			return nil
		}

		// TODO hashrate monitoring
		c.log.Infof("contract (%s) is running for %.0f seconds", c.GetID(), time.Since(*c.GetStartTime()).Seconds())

		if !c.globalScheduler.IsDeliveringAdequateHashrate(ctx, c.GetHashrateGHS(), c.GetDest(), c.hashrateDiffThreshold) {
			// cancel
			c.log.Info("Contract %s not delivering adequete hashrate", c.GetAddress())
			return fmt.Errorf("contract under delivering hashrate")
		}

		select {
		case <-ctx.Done():
			c.log.Errorf("contract context done while waiting for running contract to finish: %v", ctx.Err().Error())
			return ctx.Err()
		case <-time.After(30 * time.Second):
			continue
		}
	}
}

func (c *BTCHashrateContract) ShouldContractContinue() bool {
	return !c.ContractIsExpired() && c.ContractIsNotAvailable()
}

func (c *BTCHashrateContract) ContractIsNotAvailable() bool {
	return c.state == ContractStateRunning || c.state == ContractStatePurchased
}
