package contractmanager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/constants"
	"gitlab.com/TitanInd/hashrouter/contractmanager/contractdata"
	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
)

// BTCBuyerHashrateContract represents the collection of mining resources (collection of miners / parts of the miners) that work to fulfill single contract and monotoring tools of their performance
type BTCBuyerHashrateContract struct {
	// dependencies
	blockchain     interfaces.IBlockchainGateway
	globalHashrate interfaces.GlobalHashrate
	log            interfaces.ILogger

	// config
	data                   contractdata.ContractData
	hashrateDiffThreshold  float64
	validationBufferPeriod time.Duration
	cycleDuration          time.Duration // duration of the contract cycle that verifies the hashrate
	defaultDestination     interfaces.IDestination
	submitTimeout          time.Duration

	// state
	state                 ContractState // internal state of the contract (within hashrouter)
	fullfillmentStartedAt time.Time
}

func NewBuyerContract(
	data contractdata.ContractData,
	blockchain interfaces.IBlockchainGateway,
	globalSubmitTracker interfaces.GlobalHashrate,
	log interfaces.ILogger,
	hashrateDiffThreshold float64,
	validationBufferPeriod time.Duration,
	defaultDestination interfaces.IDestination,
	cycleDuration time.Duration,
	submitTimeout time.Duration,
) *BTCBuyerHashrateContract {
	if cycleDuration == 0 {
		cycleDuration = CYCLE_DURATION_DEFAULT
	}

	contract := &BTCBuyerHashrateContract{
		blockchain:             blockchain,
		data:                   data,
		log:                    log,
		hashrateDiffThreshold:  hashrateDiffThreshold,
		validationBufferPeriod: validationBufferPeriod,
		state:                  convertBlockchainStatusToApplicationStatus(data.State),
		defaultDestination:     defaultDestination,
		cycleDuration:          cycleDuration,
		globalHashrate:         globalSubmitTracker,
		submitTimeout:          submitTimeout,
	}

	return contract
}

func (c *BTCBuyerHashrateContract) Run(ctx context.Context) error {
	c.log.Infof("started buyer contract")

	err := c.FulfillBuyerContract(ctx)

	c.state = ContractStateAvailable
	c.log.Infof("stopped buyer contract")

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

	c.globalHashrate.Reset(c.GetWorkerName())

	// cycle checks incoming hashrate every c.cycleDuration seconds
	for {
		shouldStop, err := c.checkIteration(ctx, sub, eventsCh, ticker)
		if shouldStop {

			c.log.Infof("closing buyer contract")
			err2 := c.close(ctx)
			if err2 == nil {
				c.log.Infof("closed buyer contract")
				return err
			}

			c.log.Infof("cannot close buyer contract: %s", err2)

			// make sure cancellation will actually cancel
			if errors.Is(err, context.Canceled) {
				return err
			}
		}
	}
}

func (c *BTCBuyerHashrateContract) checkIteration(ctx context.Context, sub ethereum.Subscription, eventsCh chan types.Log, ticker *time.Ticker) (bool, error) {
	select {
	case e := <-eventsCh:
		err := c.eventsController(ctx, e)
		if err != nil {
			c.log.Errorf("blockchain event handling error: %s", err)
		}
	case err := <-sub.Err():
		return true, fmt.Errorf("contract subscription error %s", err)
	case <-ctx.Done():
		return true, ctx.Err()
	case <-ticker.C:
	}

	if c.state == ContractStatePurchased && !c.IsValidationBufferPeriod() {
		c.log.Infof("validation buffer period is over")
		c.state = ContractStateRunning
	}

	if c.ContractIsExpired() {
		c.log.Info("contract expired")
		return true, nil
	}

	if c.state == ContractStateAvailable {
		c.log.Info("contract switched to an available state")
		return true, nil
	}

	lastSubmitTime, ok := c.globalHashrate.GetLastSubmitTime(c.GetWorkerName())
	if ok {
		if time.Since(lastSubmitTime) > c.submitTimeout {
			c.log.Infof("contract last submit timeout (%s)", c.submitTimeout)
			return true, nil
		}
	}

	if !c.isDeliveringAccurateHashrate() {
		if !c.IsValidationBufferPeriod() {
			c.log.Infof("contract stopped due to delivering unaccurate hashrate after validation buffer period")
			return true, nil
		}
		c.log.Infof("contract is not delivering accurate hashrate")
	}

	c.log.Infof(
		"contract is running for %s / %s (internal/blockchain)",
		time.Since(c.fullfillmentStartedAt), time.Since(*c.GetStartTime()),
	)

	return false, nil
}

func (c *BTCBuyerHashrateContract) IsValidWallet(walletAddress common.Address) bool {
	// because buyer is not unset after contract closed it is important that only running contracts
	// are picked up by buyer (buyer field may change on every purchase unlike seller field)
	return c.data.Buyer == walletAddress && c.data.State == contractdata.ContractBlockchainStateRunning
}

func (c *BTCBuyerHashrateContract) isDeliveringAccurateHashrate() bool {
	// ignoring ok cause actualHashrate will be zero then
	averageHR, _ := c.globalHashrate.GetHashRateGHS(c.GetWorkerName())
	targetHashrateGHS := c.GetHashrateGHS()

	totalWork, _ := c.globalHashrate.GetTotalWork(c.GetWorkerName())
	wholeContractAverageGHS := hashrate.HSToGHS(hashrate.JobSubmittedToHS(float64(totalWork) / time.Since(*c.GetStartTime()).Seconds()))

	actualHashrate := wholeContractAverageGHS
	hrError := lib.RelativeError(targetHashrateGHS, actualHashrate)

	hrMsg := fmt.Sprintf(
		"worker %s, target HR %d, actual HR (9m SMA) %d, whole contract average HR %d, error %.0f%%, threshold(%.0f%%)",
		c.GetWorkerName(), targetHashrateGHS, averageHR, wholeContractAverageGHS, hrError*100, c.hashrateDiffThreshold*100,
	)

	if hrError > c.hashrateDiffThreshold {
		if actualHashrate < targetHashrateGHS {
			c.log.Warnf("contract is underdelivering: %s", hrMsg)
			return false
		}
		// contract overdelivery is fine for buyer
		c.log.Infof("contract is overdelivering: %s", hrMsg)
	} else {
		c.log.Infof("contract is delivering accurately: %s", hrMsg)
	}

	return true
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

		// setting new fulfilment start time because of validation buffer period that should
		// be applied after workername change
		c.fullfillmentStartedAt = time.Now()

		c.log.Info("updated contract destination", c.GetID())

	// Contract closed
	case blockchain.ContractClosedHex:
		c.log.Infof("received blockchain event: %s", blockchain.ContractClosedSig)

		err := c.loadBlockchainContract()
		if err != nil {
			return fmt.Errorf("cannot load blockchain contract: %s", err)
		}

		if c.data.State == contractdata.ContractBlockchainStateAvailable {
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
	if c.data.State == contractdata.ContractBlockchainStateAvailable {
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

	contractData, ok := data.(contractdata.ContractData)

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
	var hr interfaces.Hashrate
	c.globalHashrate.Range(func(m any) bool {
		workerHr := m.(*WorkerHashrateModel)
		if workerHr.ID == c.GetWorkerName() {
			hr = workerHr.hr
			return false
		}
		return true
	})
	return hr
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
	return lib.NewDest(c.GetWorkerName(), "", "your-host", nil)
}

func (c *BTCBuyerHashrateContract) GetWorkerName() string {
	return c.data.GetWorkerName()
}

func (c *BTCBuyerHashrateContract) GetStateExternal() string {
	return c.data.GetStateExternal()
}

var _ interfaces.IModel = (*BTCBuyerHashrateContract)(nil)
var _ IContractModel = (*BTCBuyerHashrateContract)(nil)
