package contractmanager

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/constants"
	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

// TODO: consider renaming to ContractInternalState to avoid collision with the state which is in blockchain
type ContractState = uint8

const (
	ContractStateAvailable ContractState = iota // contract was created and the system is following its updates
	ContractStatePurchased                      // contract was purchased but not yet picked up by miners
	ContractStateRunning                        // contract is fulfilling
)

// BTCHashrateContract represents the collection of mining resources (collection of miners / parts of the miners) that work to fulfill single contract and monotoring tools of their performance
type BTCHashrateContract struct {
	// dependencies
	blockchain      interfaces.IBlockchainGateway
	globalScheduler *GlobalSchedulerV2

	data                   blockchain.ContractData
	FullfillmentStartTime  *time.Time
	isBuyer                bool
	hashrateDiffThreshold  float64
	validationBufferPeriod time.Duration

	state ContractState // internal state of the contract (within hashrouter)

	hashrate *hashrate.Hashrate // the counter of single contract

	log                interfaces.ILogger
	stopFullfillment   chan struct{}
	defaultDestination interfaces.IDestination
}

func NewContract(
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
		isBuyer:                false,
		hashrateDiffThreshold:  hashrateDiffThreshold,
		validationBufferPeriod: validationBufferPeriod,
		globalScheduler:        globalScheduler,
		state:                  convertBlockchainStatusToApplicationStatus(data.State),
		defaultDestination:     defaultDestination,
	}

	return contract
}

func convertBlockchainStatusToApplicationStatus(status blockchain.ContractBlockchainState) ContractState {
	switch status {
	case blockchain.ContractBlockchainStateRunning:
		return ContractStatePurchased
	case blockchain.ContractBlockchainStateAvailable:
		return ContractStateAvailable
	default:
		return ContractStateAvailable
	}
}

// Runs goroutine that monitors the contract events and replace the miners which are out
func (c *BTCHashrateContract) Run(ctx context.Context) error {

	// contract was purchased before the node started, may be result of the restart
	if c.data.State == blockchain.ContractBlockchainStateRunning {
		go func() {
			time.Sleep(c.validationBufferPeriod)
			c.FulfillAndClose(ctx)
		}()
	}
	if c.isBuyer {
		// buyer node points contracts to default
		c.setDestToDefault(c.defaultDestination)
	}

	return c.listenContractEvents(ctx)
}

// Ignore checks if contract should be ignored by the node
func (c *BTCHashrateContract) IsValidWallet(walletAddress common.Address) bool {
	if c.isBuyer {
		return c.data.Buyer == walletAddress
	}

	return c.data.Seller == walletAddress
}

// Sets contract dest to default dest for buyer node
func (c *BTCHashrateContract) setDestToDefault(defaultDest interfaces.IDestination) {
	c.data.Dest = defaultDest
}

func (c *BTCHashrateContract) listenContractEvents(ctx context.Context) error {
	eventsCh, sub, err := c.blockchain.SubscribeToContractEvents(ctx, common.HexToAddress(c.GetAddress()))
	if err != nil {
		return fmt.Errorf("cannot subscribe for contract events %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			c.log.Errorf("unsubscribing from contract %v", c.GetID())
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
				c.log.Error("blockchain event handling error: %s", err)
			}
		}
	}
}

func (c *BTCHashrateContract) eventsController(ctx context.Context, eventHex string) error {
	switch eventHex {

	// Contract Purchased
	case blockchain.ContractPurchasedHex:
		err := c.LoadBlockchainContract()
		if err != nil {
			return err
		}
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

func (c *BTCHashrateContract) LoadBlockchainContract() error {
	data, err := c.blockchain.ReadContract(c.data.Addr)
	if err != nil {
		return fmt.Errorf("cannot read contract: %s, address (%s)", err, c.data.Addr)
	}
	// TODO guard it
	contractData, ok := data.(blockchain.ContractData)

	if !ok {
		return fmt.Errorf("failed to load blockhain data, address (%s)", c.data.Addr)
	}

	c.data = contractData
	c.log.Debugf("Loaded contract: %v - %v \n", c.GetID(), c.data)
	return nil
}

func (c *BTCHashrateContract) FulfillAndClose(ctx context.Context) {
	var err error
	if c.isBuyer {
		err = c.FulfillBuyerContract(ctx)
	} else {
		err = c.FulfillContract(ctx)
	}
	if err != nil {
		c.log.Errorf("error during contract fulfillment: %s", err)
		err := c.Close(ctx)
		if err != nil {
			c.log.Errorf("error during contract closeout: %s", err)
		}
	}
}

// FulfillContract fulfills contract and returns error when contract is finished (NO CLOSEOUT)
func (c *BTCHashrateContract) FulfillContract(ctx context.Context) error {
	c.state = ContractStatePurchased

	if c.ContractIsExpired() {
		c.log.Warn("contract is expired %s", c.GetID())
		return fmt.Errorf("contract is expired")
	}

	c.stopFullfillment = make(chan struct{}, 10)

	// running cycle checks combination every N seconds
	for {
		if c.ContractIsExpired() {
			c.log.Info("contract time ended, or state is closed, closing...", c.GetID())
			return fmt.Errorf("contract is expired")
		}

		// TODO hashrate monitoring
		c.log.Infof("contract (%s) is running for %.0f seconds", c.GetID(), time.Since(*c.GetStartTime()).Seconds())

		err := c.globalScheduler.Update(c.GetID(), c.GetHashrateGHS(), c.GetDest())
		if err != nil {
			c.log.Errorf("cannot update combination %s", err)
		}

		if c.state != ContractStateRunning {
			c.state = ContractStateRunning
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

func (c *BTCHashrateContract) Close(ctx context.Context) error {
	c.log.Debugf("closing contract %v", c.GetID())
	c.Stop(ctx)

	closeoutAccount := c.GetCloseoutAccount()

	err := c.blockchain.SetContractCloseOut(closeoutAccount, c.GetAddress(), int64(c.GetCloseoutType()))
	if err != nil {
		c.log.Error("cannot close contract", err)
		return err
	}
	c.state = ContractStateAvailable
	return nil
}

// Stops fulfilling the contract by miners
func (c *BTCHashrateContract) Stop(ctx context.Context) {
	c.log.Infof("Attempting to stop contract %v; with state %v", c.GetID(), c.state)
	if c.state == ContractStateRunning {
		c.log.Infof("Stopping contract %v", c.GetID())

		err := c.globalScheduler.Update(c.GetID(), 0, c.GetDest())
		if err != nil {
			c.log.Error(err)
		}

		c.state = ContractStateAvailable

		if c.stopFullfillment != nil {
			c.stopFullfillment <- struct{}{}
		}
		return
	}

	c.log.Warnf("contract (%s) is not running", c.GetID())
}

func (c *BTCHashrateContract) ContractIsExpired() bool {
	endTime := c.GetEndTime()
	if endTime == nil {
		return false
	}
	return time.Now().After(*endTime)
}

func (c *BTCHashrateContract) GetBuyerAddress() string {
	return c.data.Buyer.String()
}

func (c *BTCHashrateContract) GetSellerAddress() string {
	return c.data.Seller.String()
}

func (c *BTCHashrateContract) GetID() string {
	return c.GetAddress()
}

func (c *BTCHashrateContract) GetAddress() string {
	return c.data.Addr.String()
}

func (c *BTCHashrateContract) GetHashrateGHS() int {
	return int(c.data.Speed / int64(math.Pow10(9)))
}

func (c *BTCHashrateContract) GetDuration() time.Duration {
	return time.Duration(c.data.Length) * time.Second
}

func (c *BTCHashrateContract) GetStartTime() *time.Time {
	startTime := time.Unix(c.data.StartingBlockTimestamp, 0)

	return &startTime
}

func (c *BTCHashrateContract) GetEndTime() *time.Time {
	endTime := c.GetStartTime().Add(c.GetDuration())
	return &endTime
}

func (c *BTCHashrateContract) GetState() ContractState {
	return c.state
}

func (c *BTCHashrateContract) GetDest() interfaces.IDestination {
	return c.data.Dest
}

func (c *BTCHashrateContract) GetCloseoutType() constants.CloseoutType {
	if c.isBuyer {
		return constants.CloseoutTypeCancel
	}
	return constants.CloseoutTypeWithoutClaim
}

func (c *BTCHashrateContract) GetCloseoutAccount() string {
	if c.isBuyer {
		return c.GetBuyerAddress()
	}
	return c.GetSellerAddress()
}

func (c *BTCHashrateContract) GetStatusInternal() string {
	return c.data.State.String()
}

var _ interfaces.IModel = (*BTCHashrateContract)(nil)
