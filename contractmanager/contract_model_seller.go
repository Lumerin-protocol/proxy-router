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

// BTCHashrateContractSeller represents the collection of mining resources (collection of miners / parts of the miners) that work to fulfill single contract and monotoring tools of their performance
type BTCHashrateContractSeller struct {
	// dependencies
	blockchain      interfaces.IBlockchainGateway
	globalScheduler interfaces.IGlobalScheduler
	log             interfaces.ILogger

	// config
	cycleDuration          time.Duration // duration of the contract cycle that verifies the hashrate
	defaultDestination     interfaces.IDestination
	hashrateDiffThreshold  float64
	isBuyer                bool
	validationBufferPeriod time.Duration

	// internal state
	data                  blockchain.ContractData
	FullfillmentStartTime *time.Time
	state                 ContractState      // internal state of the contract (within hashrouter)
	hashrate              *hashrate.Hashrate // the counter of single contract
	stopFullfillment      chan struct{}
}

func NewContract(
	data blockchain.ContractData,
	blockchain interfaces.IBlockchainGateway,
	globalScheduler interfaces.IGlobalScheduler,
	log interfaces.ILogger,
	hr *hashrate.Hashrate,
	hashrateDiffThreshold float64,
	validationBufferPeriod time.Duration,
	defaultDestination interfaces.IDestination,
	cycleDuration time.Duration,
) *BTCHashrateContractSeller {

	if hr == nil {
		hr = hashrate.NewHashrateV2(hashrate.NewSma(9 * time.Minute))
	}

	if cycleDuration == 0 {
		cycleDuration = CYCLE_DURATION_DEFAULT
	}

	contract := &BTCHashrateContractSeller{
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
		cycleDuration:          cycleDuration,
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
func (c *BTCHashrateContractSeller) Run(ctx context.Context) error {

	// contract was purchased before the node started, may be result of the restart
	if c.data.State == blockchain.ContractBlockchainStateRunning {
		go func() {
			c.FulfillAndClose(ctx)
		}()
	}

	return c.listenContractEvents(ctx)
}

// Ignore checks if contract should be ignored by the node
func (c *BTCHashrateContractSeller) IsValidWallet(walletAddress common.Address) bool {
	return c.data.Seller == walletAddress
}

func (c *BTCHashrateContractSeller) listenContractEvents(ctx context.Context) error {
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

func (c *BTCHashrateContractSeller) eventsController(ctx context.Context, eventHex string) error {
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
	case blockchain.ContractClosedHex:
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

func (c *BTCHashrateContractSeller) LoadBlockchainContract() error {
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
	c.log.Debugf("loaded contract: %s \n %v ", c.GetID(), c.data)
	return nil
}

func (c *BTCHashrateContractSeller) FulfillAndClose(ctx context.Context) {
	err := c.FulfillContract(ctx)
	c.log.Infof("contract(%s) fulfillment finished: %s", c.GetID(), err)

	err = c.Close(ctx)
	if err != nil {
		c.log.Errorf("error during contract closeout: %s", err)
	} else {
		c.log.Infof("contract(%s) closed", c.GetID())
	}
	c.hashrate = hashrate.NewHashrateV2(hashrate.NewSma(9 * time.Minute))
}

// FulfillContract fulfills contract and returns error when contract is finished (NO CLOSEOUT)
func (c *BTCHashrateContractSeller) FulfillContract(ctx context.Context) error {
	c.state = ContractStatePurchased

	if c.ContractIsExpired() {
		return fmt.Errorf("contract is expired %s", c.GetID())
	}

	c.stopFullfillment = make(chan struct{}, 10)

	// running cycle checks combination every N seconds
	for {
		if c.ContractIsExpired() {
			return fmt.Errorf("contract is expired: %s", c.GetID())
		}

		// TODO hashrate monitoring
		c.log.Infof("contract (%s) is running for %s seconds, dest %s", c.GetID(), time.Since(*c.GetStartTime()), c.GetDest())

		err := c.globalScheduler.Update(c.GetID(), c.GetHashrateGHS(), c.GetDest(), c.hashrate)
		if err != nil {
			c.log.Errorf("cannot update combination %s", err)
		}

		if c.state != ContractStateRunning {
			c.state = ContractStateRunning
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopFullfillment:
			return fmt.Errorf("contract was stopped: %s", c.GetID())
		case <-time.After(c.cycleDuration):
			continue
		}
	}
}

func (c *BTCHashrateContractSeller) Close(ctx context.Context) error {
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

// Stops fulfilling the contract by miners
func (c *BTCHashrateContractSeller) Stop(ctx context.Context) {
	c.log.Infof("attempting to stop contract %v; with state %v", c.GetID(), c.state)
	if c.state == ContractStateRunning {
		c.log.Infof("Stopping contract %v", c.GetID())

		// TODO: deallocation should be performed within main loop
		// and this function should only send a signal to stop
		err := c.globalScheduler.Update(c.GetID(), 0, c.GetDest(), nil)
		if err != nil {
			c.log.Error(err)
		}

		c.state = ContractStateAvailable

		if c.stopFullfillment != nil {
			c.stopFullfillment <- struct{}{}
		}
		return
	} else {
		c.log.Warnf("contract already (%s) stopped")
	}

}

func (c *BTCHashrateContractSeller) ContractIsExpired() bool {
	endTime := c.GetEndTime()
	if endTime == nil {
		return false
	}
	return time.Now().After(*endTime)
}

func (c *BTCHashrateContractSeller) GetBuyerAddress() string {
	return c.data.Buyer.String()
}

func (c *BTCHashrateContractSeller) GetSellerAddress() string {
	return c.data.Seller.String()
}

func (c *BTCHashrateContractSeller) GetID() string {
	return c.GetAddress()
}

func (c *BTCHashrateContractSeller) GetAddress() string {
	return c.data.Addr.String()
}

func (c *BTCHashrateContractSeller) GetHashrateGHS() int {
	return int(c.data.Speed / int64(math.Pow10(9)))
}

func (c *BTCHashrateContractSeller) GetDuration() time.Duration {
	return time.Duration(c.data.Length) * time.Second
}

func (c *BTCHashrateContractSeller) GetStartTime() *time.Time {
	startTime := time.Unix(c.data.StartingBlockTimestamp, 0)
	return &startTime
}

func (c *BTCHashrateContractSeller) GetEndTime() *time.Time {
	endTime := c.GetStartTime().Add(c.GetDuration())
	return &endTime
}

func (c *BTCHashrateContractSeller) GetState() ContractState {
	return c.state
}

func (c *BTCHashrateContractSeller) GetDest() interfaces.IDestination {
	return c.data.Dest
}

func (c *BTCHashrateContractSeller) SetDest(dest interfaces.IDestination) {
	c.data.Dest = dest
}

func (c *BTCHashrateContractSeller) GetCloseoutType() constants.CloseoutType {
	return constants.CloseoutTypeWithoutClaim
}

func (c *BTCHashrateContractSeller) GetStatusInternal() string {
	return c.data.State.String()
}

func (c *BTCHashrateContractSeller) GetDeliveredHashrate() interfaces.Hashrate {
	return c.hashrate
}

func (c *BTCHashrateContractSeller) IsBuyer() bool {
	return false
}

var _ interfaces.IModel = (*BTCHashrateContractSeller)(nil)
var _ IContractModel = (*BTCHashrateContractSeller)(nil)
