package contract

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
	hashrateContract "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
)

type ValidationState int8

const (
	ValidationStateWaitingFirstShare ValidationState = 0
	ValidationStateNotValidating     ValidationState = 1
	ValidationStateValidating        ValidationState = 2
	ValidationStateFinished          ValidationState = 3
)

type ContractWatcherBuyer struct {
	// config
	firstShareTimeout      time.Duration // time to wait for the first share to arrive, otherwise close contract
	validationStartTimeout time.Duration // time when validation kicks in
	shareTimeout           time.Duration // time to wait for the share to arrive, otherwise close contract
	hrErrorThreshold       float64       // hashrate relative error threshold for the contract to be considered fulfilling accurately

	terms                 *hashrateContract.EncryptedTerms
	state                 resources.ContractState
	validationState       ValidationState
	fulfillmentStartedAt  *time.Time
	contractCycleDuration time.Duration

	tsk *lib.Task

	//deps
	allocator      *allocator.Allocator
	globalHashrate *hashrate.GlobalHashrate
	log            interfaces.ILogger
}

func NewContractWatcherBuyer(data *hashrateContract.EncryptedTerms, cycleDuration time.Duration, hashrateFactory func() *hashrate.Hashrate, allocator *allocator.Allocator, log interfaces.ILogger) *ContractWatcherBuyer {
	return &ContractWatcherBuyer{
		terms:                 data,
		state:                 resources.ContractStatePending,
		allocator:             allocator,
		contractCycleDuration: cycleDuration,
		log:                   log,
	}
}

func (p *ContractWatcherBuyer) StartFulfilling(ctx context.Context) {
	p.log.Infof("buyer contract started fulfilling")

	p.tsk = lib.NewTaskFunc(p.Run)
	p.tsk.Start(ctx)
}

func (p *ContractWatcherBuyer) StopFulfilling() {
	<-p.tsk.Stop()
	p.log.Infof("buyer contract stopped fulfilling")
}

func (p *ContractWatcherBuyer) Done() <-chan struct{} {
	return p.tsk.Done()
}

func (p *ContractWatcherBuyer) Err() error {
	if errors.Is(p.tsk.Err(), context.Canceled) {
		return ErrContractClosed
	}
	return p.tsk.Err()
}

func (p *ContractWatcherBuyer) SetData(data *hashrateContract.EncryptedTerms) {
	p.terms = data
}

func (p *ContractWatcherBuyer) Run(ctx context.Context) error {
	p.state = resources.ContractStateRunning
	startedAt := time.Now()
	p.fulfillmentStartedAt = &startedAt

	// instead of resetting write a method that creates separate counters for each worker at given moment of time
	p.globalHashrate.Reset(p.terms.ContractID)

	ticker := time.NewTicker(p.contractCycleDuration)
	defer ticker.Stop()

	endTimer := time.Timer{}
	for {
		endTimer.Reset(time.Until(*p.GetEndTime()))
		select {
		case <-ctx.Done():
			if !endTimer.Stop() {
				<-endTimer.C
			}
			return ctx.Err()
		case <-endTimer.C:
			return nil
		case <-ticker.C:
			if !endTimer.Stop() {
				<-endTimer.C
			}

			err := p.checkIncomingHashrate(ctx)
			if err != nil {
				return err
			}
		}
	}
}

func (p *ContractWatcherBuyer) checkIncomingHashrate(ctx context.Context) error {
	if p.validationState == ValidationStateWaitingFirstShare && p.isFirstShareTimeout() {
		p.validationState = ValidationStateNotValidating
	}

	if p.validationState == ValidationStateNotValidating && p.isValidationStartTimeout() {
		p.validationState = ValidationStateValidating
	}

	if p.isContractExpired() {
		p.validationState = ValidationStateFinished
	}

	isHashrateOK := p.isReceivingAcceptableHashrate()

	switch p.validationState {
	case ValidationStateWaitingFirstShare:
		// no validation
		return nil
	case ValidationStateNotValidating:
		// on this stage there should be at least one share
		lastShareTime, ok := p.globalHashrate.GetLastSubmitTime(p.getWorkerName())
		if !ok {
			return fmt.Errorf("no share submitted within firstShareTimeout (%s)", p.firstShareTimeout)
		}
		if time.Since(lastShareTime) > p.shareTimeout {
			return fmt.Errorf("no share submitted within shareTimeout (%s)", p.shareTimeout)
		}
		return nil
	case ValidationStateValidating:
		lastShareTime, ok := p.globalHashrate.GetLastSubmitTime(p.getWorkerName())
		if !ok {
			errMsg := "on ValidationStateValidating there should be at least one share"
			p.log.DPanic(errMsg)
			return fmt.Errorf(errMsg)
		}
		if time.Since(lastShareTime) > p.shareTimeout {
			return fmt.Errorf("no share submitted within shareTimeout (%s)", p.shareTimeout)
		}
		if !isHashrateOK {
			return fmt.Errorf("contract is not delivering accurate hashrate")
		}
		return nil
	case ValidationStateFinished:
		return fmt.Errorf("contract is finished")
	default:
		return fmt.Errorf("unknown validation state")
	}
}

func (p *ContractWatcherBuyer) isReceivingAcceptableHashrate() bool {
	// ignoring ok cause actualHashrate will be zero then
	actualHashrate, _ := p.globalHashrate.GetHashRateGHS(p.getWorkerName(), "mean")
	targetHashrateGHS := p.GetHashrateGHS()

	hrError := lib.RelativeError(targetHashrateGHS, actualHashrate)

	hrMsg := fmt.Sprintf(
		"worker %s, target HR %.0f, whole contract average HR %.0f, error %.0f%%, threshold(%.0f%%)",
		p.getWorkerName(), targetHashrateGHS, actualHashrate, hrError*100, p.hrErrorThreshold*100,
	)

	if hrError < p.hrErrorThreshold {
		p.log.Infof("contract is delivering accurately: %s", hrMsg)
		return true
	}

	if actualHashrate > targetHashrateGHS {
		p.log.Infof("contract is overdelivering: %s", hrMsg)
		// contract overdelivery is fine for buyer
		return true
	}

	p.log.Warnf("contract is underdelivering: %s", hrMsg)
	return false
}

func (p *ContractWatcherBuyer) GetRole() resources.ContractRole {
	return resources.ContractRoleBuyer
}

func (p *ContractWatcherBuyer) GetDest() string {
	return ""
}

func (p *ContractWatcherBuyer) GetDuration() time.Duration {
	return p.terms.Duration
}

func (p *ContractWatcherBuyer) GetEndTime() *time.Time {
	if p.terms.StartsAt == nil {
		return nil
	}
	endTime := p.terms.StartsAt.Add(p.terms.Duration)
	return &endTime
}

func (p *ContractWatcherBuyer) GetFulfillmentStartedAt() *time.Time {
	return p.fulfillmentStartedAt
}

func (p *ContractWatcherBuyer) GetID() string {
	return p.terms.ContractID
}

func (p *ContractWatcherBuyer) GetHashrateGHS() float64 {
	return p.terms.Hashrate
}

// func (p *ContractWatcher) GetResourceEstimates() map[string]float64 {
// 	return p.data.ResourceEstimates
// }

func (p *ContractWatcherBuyer) GetResourceEstimatesActual() map[string]float64 {
	res, _ := p.globalHashrate.GetHashRateGHSAll(p.getWorkerName())
	return res
}

func (p *ContractWatcherBuyer) GetResourceType() string {
	return ResourceTypeHashrate
}

func (p *ContractWatcherBuyer) GetSeller() string {
	return p.terms.Seller
}

func (p *ContractWatcherBuyer) GetBuyer() string {
	return p.terms.Buyer
}

func (p *ContractWatcherBuyer) GetStartedAt() *time.Time {
	return p.terms.StartsAt
}

func (p *ContractWatcherBuyer) GetState() resources.ContractState {
	return p.state
}

func (p *ContractWatcherBuyer) isFirstShareTimeout() bool {
	return time.Since(*p.fulfillmentStartedAt) > p.firstShareTimeout
}

func (p *ContractWatcherBuyer) isValidationStartTimeout() bool {
	return time.Since(*p.fulfillmentStartedAt) > p.validationStartTimeout
}

func (p *ContractWatcherBuyer) isContractExpired() bool {
	endTime := p.GetEndTime()
	if endTime == nil {
		return false
	}
	return time.Now().After(*endTime)
}

func (p *ContractWatcherBuyer) getWorkerName() string {
	return p.terms.ContractID
}
