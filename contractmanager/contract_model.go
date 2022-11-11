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
	"gitlab.com/TitanInd/hashrouter/lib"
)

// TODO: consider renaming to ContractInternalState to avoid collision with the state which is in blockchain
type ContractState = uint8

const (
	ContractStateAvailable ContractState = iota // contract was created and the system is following its updates
	ContractStatePurchased                      // contract was purchased but not yet picked up by miners
	ContractStateRunning                        // contract is fulfilling
	ContractStateClosed                         // contract is closed
)

// BTCHashrateContract represents the collection of mining resources (collection of miners / parts of the miners) that work to fulfill single contract and monotoring tools of their performance
type BTCHashrateContract struct {
	// dependencies
	blockchain      interfaces.IBlockchainGateway
	globalScheduler *GlobalSchedulerService

	data                   blockchain.ContractData
	FullfillmentStartTime  *time.Time
	isBuyer                bool
	hashrateDiffThreshold  float64
	validationBufferPeriod time.Duration

	state ContractState // internal state of the contract (within hashrouter)

	hashrate *hashrate.Hashrate // the counter of single contract
	minerIDs []string           // miners involved in fulfilling this contract

	log              interfaces.ILogger
	stopFullfillment chan struct{}
}

func NewContract(
	data blockchain.ContractData,
	blockchain interfaces.IBlockchainGateway,
	globalScheduler *GlobalSchedulerService,
	log interfaces.ILogger,
	hr *hashrate.Hashrate,
	isBuyer bool,
	hashrateDiffThreshold float64,
	validationBufferPeriod time.Duration,
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
	}

	return contract
}

func convertBlockchainStatusToApplicationStatus(status blockchain.ContractBlockchainState) ContractState {
	switch status {
	case blockchain.ContractBlockchainStateRunning:
		return ContractStateRunning
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

	return c.listenContractEvents(ctx)
}

// Ignore checks if contract should be ignored by the node
func (c *BTCHashrateContract) Ignore(walletAddress common.Address, defaultDest lib.Dest) bool {
	if c.isBuyer {
		if c.data.Buyer != walletAddress {
			return true
		}
		// buyer node points contracts to default
		c.setDestToDefault(defaultDest)
		return false
	}

	if c.data.Seller != walletAddress {
		return true
	}
	return false
}

// Sets contract dest to default dest for buyer node
func (c *BTCHashrateContract) setDestToDefault(defaultDest lib.Dest) {
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

		c.globalScheduler.DeallocateContract(ctx, c.GetID())

		err = c.StartHashrateAllocation()
		if err != nil {
			return fmt.Errorf("cannot start hashrate allocation for ContractCipherTextUpdated event: %w", err)
		}

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
	err := c.FulfillContract(ctx)
	if err != nil {
		c.log.Errorf("error during contract fulfillment: %s", err)
		err := c.Close(ctx)
		if err != nil {
			c.log.Errorf("error during contract closeout: %s", err)
		}
	}
}

func (c *BTCHashrateContract) FulfillContract(ctx context.Context) error {
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

			minerIDs, err := c.globalScheduler.UpdateCombination(ctx, c.minerIDs, c.GetHashrateGHS(), c.GetDest(), c.GetID(), c.hashrateDiffThreshold)
			if err != nil {
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

func (c *BTCHashrateContract) StartHashrateAllocation() error {
	c.state = ContractStateRunning

	minerList, err := c.globalScheduler.Allocate(c.GetID(), c.GetHashrateGHS(), c.data.Dest)
	if err != nil {
		return err
	}

	c.minerIDs = minerList.IDs()

	c.log.Infof("fulfilling contract %s; expires at %v", c.GetID(), c.GetEndTime())

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
	if c.ContractIsNotAvailable() {

		c.log.Infof("Stopping contract %v", c.GetID())

		c.globalScheduler.DeallocateContract(ctx, c.GetID())

		c.state = ContractStateAvailable
		c.minerIDs = []string{}

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

func (c *BTCHashrateContract) ShouldContractContinue() bool {
	// c.log.Infof("is the contract expired? %v", c.ContractIsExpired())
	// c.log.Infof("is the contract available? %v", !c.ContractIsNotAvailable())
	return !c.ContractIsExpired() && c.ContractIsNotAvailable()
}

func (c *BTCHashrateContract) ContractIsNotAvailable() bool {
	return c.state == ContractStateRunning || c.state == ContractStatePurchased
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

func (c *BTCHashrateContract) GetDest() lib.Dest {
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
