package contractmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/constants"
	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

// BTCBuyerHashrateContract represents the collection of mining resources (collection of miners / parts of the miners) that work to fulfill single contract and monotoring tools of their performance
type BTCBuyerHashrateContract struct {
	// dependencies
	blockchain      interfaces.IBlockchainGateway
	globalScheduler interfaces.IGlobalScheduler
	hashrate        *hashrate.Hashrate // the counter of single contract
	log             interfaces.ILogger

	// config
	data                   blockchain.ContractData
	hashrateDiffThreshold  float64
	validationBufferPeriod time.Duration
	cycleDuration          time.Duration // duration of the contract cycle that verifies the hashrate
	defaultDestination     interfaces.IDestination

	// state
	state                 ContractState // internal state of the contract (within hashrouter)
	fullfillmentStartedAt time.Time
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
		hr = hashrate.NewHashrateV2(hashrate.NewSma(9 * time.Minute))
	}

	if cycleDuration == 0 {
		cycleDuration = CYCLE_DURATION_DEFAULT
	}

	contract := &BTCBuyerHashrateContract{
		blockchain:             blockchain,
		data:                   data,
		hashrate:               hr,
		log:                    log,
		hashrateDiffThreshold:  hashrateDiffThreshold,
		validationBufferPeriod: validationBufferPeriod,
		globalScheduler:        globalScheduler,
		state:                  convertBlockchainStatusToApplicationStatus(data.State),
		defaultDestination:     defaultDestination,
		cycleDuration:          cycleDuration,
	}
	return contract
}

func (c *BTCBuyerHashrateContract) Run(ctx context.Context) error {
	c.log.Infof("started buyer contract")

	err := c.FulfillBuyerContract(ctx)

	c.state = ContractStateAvailable

	c.log.Infof("stopped buyer contract")

	err2 := c.close(ctx)
	if err2 != nil {
		return err2
	}

	return err
}

func (c *BTCBuyerHashrateContract) FulfillBuyerContract(ctx context.Context) error {
	c.log.Debugf("started fulfilment of the buyer contract, validation buffer period (%s)", c.validationBufferPeriod)

	c.state = ContractStatePurchased
	c.fullfillmentStartedAt = time.Now()

	eventsCh, sub, err := c.blockchain.SubscribeToContractEvents(ctx, common.HexToAddress(c.GetAddress()))
	if err != nil {
		return fmt.Errorf("cannot subscribe for contract events %w", err)
	}
	defer sub.Unsubscribe()

	ticker := time.NewTicker(c.cycleDuration)
	defer ticker.Stop()

	// cycle checks incoming hashrate every c.cycleDuration seconds
	for {
		if c.state == ContractStatePurchased && !c.IsValidationBufferPeriod() {
			c.log.Infof("validation buffer period is over")
			c.state = ContractStateRunning
		}

		if c.ContractIsExpired() {
			c.log.Info("contract expired")
			return nil
		}

		if c.state == ContractStateAvailable {
			c.log.Info("contract switched to an available state")
			return nil
		}

		if !c.globalScheduler.IsDeliveringAdequateHashrate(ctx, c.GetHashrateGHS(), c.GetDest(), c.hashrateDiffThreshold) {
			c.log.Infof("contract is not delivering adequate hashrate")
			if !c.IsValidationBufferPeriod() {
				return nil
			}
		}

		c.log.Infof("contract is running for %s / %s (internal/blockchain)", time.Since(c.fullfillmentStartedAt), time.Since(*c.GetStartTime()))

		select {
		case e := <-eventsCh:
			err := c.eventsController(ctx, e)
			if err != nil {
				c.log.Errorf("blockchain event handling error: %s", err)
			}
		case err := <-sub.Err():
			return fmt.Errorf("contract subscription error %s", err)
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			continue
		}
	}
}

func (c *BTCBuyerHashrateContract) IsValidWallet(walletAddress common.Address) bool {
	// because buyer is not unset after contract closed it is important that only running contracts
	// are picked up by buyer (buyer field may change on every purchase unlike seller field)
	return c.data.Buyer == walletAddress && c.data.State == blockchain.ContractBlockchainStateRunning
}

func (c *BTCBuyerHashrateContract) getCloseoutType() constants.CloseoutType {
	return constants.CloseoutTypeCancel
}

func (c *BTCBuyerHashrateContract) eventsController(ctx context.Context, e types.Log) error {
	eventHex := e.Topics[0].Hex()

	switch eventHex {
	// Contract updated on buyer side
	case blockchain.ContractCipherTextUpdatedHex:
		c.log.Infof("received blockchain event: %s", blockchain.ContractCipherTextUpdatedSig)

		// updated destination will be picked up on the next miner cycle
		err := c.loadBlockchainContract()
		if err != nil {
			return err
		}

		c.log.Info("updated contract destination", c.GetDest())

	// Contract closed
	case blockchain.ContractClosedHex:
		c.log.Infof("received blockchain event: %s", blockchain.ContractClosedSig)

		err := c.loadBlockchainContract()
		if err != nil {
			return fmt.Errorf("cannot load blockchain contract: %s", err)
		}

		if c.data.State == blockchain.ContractBlockchainStateAvailable {
			c.state = ContractStateAvailable
		}

	// not relevant, contract model buyer picks only already purchased contracts and exits on closeout
	case blockchain.ContractPurchasedHex:
		c.log.Infof("received blockchain event: %s", blockchain.ContractPurchasedSig)

	case blockchain.ContractPurchaseInfoUpdatedHex:
		c.log.Infof("received blockchain event: %s", blockchain.ContractPurchaseInfoUpdatedSig)
	// Contract updated on seller side - not relevant because it can be update only when contract is not running

	default:
		c.log.Info("received unknown blockchain event %s", eventHex)
	}

	return nil
}

func (c *BTCBuyerHashrateContract) close(ctx context.Context) error {
	if c.data.State == blockchain.ContractBlockchainStateAvailable {
		c.log.Debugf("contract already closed")
		return nil
	}

	c.log.Debugf("closing contract")

	err := c.blockchain.SetContractCloseOut(c.GetAddress(), int64(c.getCloseoutType()))
	if err != nil {
		c.log.Error("cannot close contract", err)
		return err
	}

	c.log.Debugf("contract closed")
	return nil
}

func (c *BTCBuyerHashrateContract) loadBlockchainContract() error {
	data, err := c.blockchain.ReadContract(c.data.Addr)
	if err != nil {
		return fmt.Errorf("cannot read contract: %s, address (%s)", err, c.data.Addr)
	}

	contractData, ok := data.(blockchain.ContractData)

	if !ok {
		return fmt.Errorf("failed to load blockhain data, address (%s)", c.data.Addr)
	}

	c.data = contractData
	c.log.Debugf("loaded contract: %#+v ", c.data)
	return nil
}

func (c *BTCBuyerHashrateContract) GetID() string {
	return c.GetAddress()
}

func (c *BTCBuyerHashrateContract) GetDeliveredHashrate() interfaces.Hashrate {
	return c.hashrate
}

func (c *BTCBuyerHashrateContract) GetState() ContractState {
	return c.state
}

func (c *BTCBuyerHashrateContract) IsBuyer() bool {
	return true
}

func (c *BTCBuyerHashrateContract) IsValidationBufferPeriod() bool {
	return time.Since(c.fullfillmentStartedAt) < c.validationBufferPeriod
}

func (c *BTCBuyerHashrateContract) FullfillmentStartedAt() time.Time {
	return c.fullfillmentStartedAt
}

// all following methods are delegated to c.data

func (c *BTCBuyerHashrateContract) ContractIsExpired() bool {
	return c.data.ContractIsExpired()
}

func (c *BTCBuyerHashrateContract) GetBuyerAddress() string {
	return c.data.GetBuyerAddress()
}

func (c *BTCBuyerHashrateContract) GetSellerAddress() string {
	return c.data.GetSellerAddress()
}

func (c *BTCBuyerHashrateContract) GetAddress() string {
	return c.data.GetAddress()
}

func (c *BTCBuyerHashrateContract) GetHashrateGHS() int {
	return c.data.GetHashrateGHS()
}

func (c *BTCBuyerHashrateContract) GetDuration() time.Duration {
	return c.data.GetDuration()
}

func (c *BTCBuyerHashrateContract) GetStartTime() *time.Time {
	return c.data.GetStartTime()
}

func (c *BTCBuyerHashrateContract) GetEndTime() *time.Time {
	return c.data.GetEndTime()
}

func (c *BTCBuyerHashrateContract) GetDest() interfaces.IDestination {
	return c.data.GetDest()
}

func (c *BTCBuyerHashrateContract) GetStatusInternal() string {
	return c.data.GetStatusInternal()
}

var _ interfaces.IModel = (*BTCHashrateContractSeller)(nil)
var _ IContractModel = (*BTCHashrateContractSeller)(nil)
