package contractmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/constants"
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
	globalScheduler interfaces.IGlobalScheduler,
	log interfaces.ILogger,
	hr *hashrate.Hashrate,
	hashrateDiffThreshold float64,
	validationBufferPeriod time.Duration,
	defaultDestination interfaces.IDestination,
	cycleDuration time.Duration,
) *BTCBuyerHashrateContract {

	if hr == nil {
		hr = hashrate.NewHashrate()
	}

	contract := &BTCBuyerHashrateContract{
		&BTCHashrateContract{
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
			cycleDuration:          cycleDuration,
		},
	}

	return contract
}

func (c *BTCBuyerHashrateContract) Run(ctx context.Context) error {
	// contract was purchased before the node started, may be result of the restart
	if c.data.State == blockchain.ContractBlockchainStateRunning {
		// TODO: refactor to ensure there is only one fulfillAndClose goroutine per instance
		go c.FulfillAndClose(ctx)
	}
	// buyer node points contracts to default
	c.setDestToDefault(c.defaultDestination)

	return c.listenContractEvents(ctx)
}

func (c *BTCBuyerHashrateContract) FulfillBuyerContract(ctx context.Context) error {
	c.state = ContractStatePurchased
	c.log.Debugf("waiting for validation buffer period (%s) for contract %s", c.validationBufferPeriod, c.GetID())
	time.Sleep(c.validationBufferPeriod)

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
		case <-time.After(c.cycleDuration):
			continue
		}
	}
}

func (c *BTCBuyerHashrateContract) ShouldContractContinue() bool {
	return !c.ContractIsExpired() && c.ContractIsNotAvailable()
}

func (c *BTCBuyerHashrateContract) ContractIsNotAvailable() bool {
	return c.state == ContractStateRunning || c.state == ContractStatePurchased
}

func (c *BTCBuyerHashrateContract) FulfillAndClose(ctx context.Context) {
	err := c.FulfillBuyerContract(ctx)

	if err != nil {
		c.log.Errorf("error during contract fulfillment: %s", err)
		err := c.Close(ctx)
		if err != nil {
			c.log.Errorf("error during contract closeout: %s", err)
		}
	}
}

func (c *BTCBuyerHashrateContract) IsValidWallet(walletAddress common.Address) bool {
	return c.data.Buyer == walletAddress
}

func (c *BTCBuyerHashrateContract) GetCloseoutType() constants.CloseoutType {
	return constants.CloseoutTypeCancel
}

func (c *BTCBuyerHashrateContract) listenContractEvents(ctx context.Context) error {
	eventsCh, sub, err := c.blockchain.SubscribeToContractEvents(ctx, common.HexToAddress(c.GetAddress()))
	if err != nil {
		return fmt.Errorf("cannot subscribe for contract events %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			c.log.Warnf("context cancelled: unsubscribing from contract %v", c.GetID())
			sub.Unsubscribe()
			return ctx.Err()
		case err := <-sub.Err():
			c.log.Errorf("contract subscription error %v %s", c.GetID(), err)
			sub.Unsubscribe()
			return err
		case e := <-eventsCh:
			eventHex := e.Topics[0].Hex()
			err := c.eventsController(ctx, eventHex)
			if err != nil {
				c.log.Errorf("blockchain event handling error: %s", err)
			}
		}
	}
}

func (c *BTCBuyerHashrateContract) eventsController(ctx context.Context, eventHex string) error {
	switch eventHex {

	// Contract Purchased
	case blockchain.ContractPurchasedHex:
		err := c.LoadBlockchainContract()
		if err != nil {
			return err
		}
		// TODO: check if already fulfilling
		go c.FulfillAndClose(ctx)

	// Contract updated on buyer side
	case blockchain.ContractCipherTextUpdatedHex:
		c.log.Info("received contract ContractCipherTextUpdated event", c.data.Addr)
		err := c.LoadBlockchainContract()
		if err != nil {
			return err
		}
		c.log.Info("ContractCipherTextUpdated new destination", c.GetDest())
		// updated destination will be picked up on the next miner cycle

	// Contract updated on seller side
	case blockchain.ContractPurchaseInfoUpdatedHex:
		c.log.Info("received contract ContractPurchaseInfoUpdated event", c.data.Addr)

		err := c.LoadBlockchainContract()
		if err != nil {
			return fmt.Errorf("cannot load blockchain contract: %w", err)
		}

	// Contract closed
	case blockchain.ContractClosedSigHex:
		c.log.Info("received contract closed event ", c.data.Addr)

		err := c.LoadBlockchainContract()
		if err != nil {
			return fmt.Errorf("cannot load blockchain contract: %w", err)
		}

		// TODO: update internal state and let the control flow be handled in contract cycle
		c.Stop(ctx)

	default:
		c.log.Info("received unknown blockchain event %s", eventHex)
	}

	return nil
}

func (c *BTCBuyerHashrateContract) Close(ctx context.Context) error {
	c.log.Debugf("closing contract %v", c.GetID())
	c.Stop(ctx)

	err := c.blockchain.SetContractCloseOut(c.GetAddress(), int64(c.GetCloseoutType()))
	if err != nil {
		c.log.Error("cannot close contract", err)
		return err
	}
	c.state = ContractStateAvailable
	return nil
}
