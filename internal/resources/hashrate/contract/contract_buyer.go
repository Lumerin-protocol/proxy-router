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

type ContractWatcherBuyer struct {
	// config
	contractCycleDuration  time.Duration
	validationStartTimeout time.Duration // time when validation kicks in
	shareTimeout           time.Duration // time to wait for the share to arrive, otherwise close contract
	hrErrorThreshold       float64       // hashrate relative error threshold for the contract to be considered fulfilling accurately

	terms                *hashrateContract.EncryptedTerms
	state                resources.ContractState
	validationStage      ValidationStage
	fulfillmentStartedAt *time.Time

	tsk *lib.Task

	//deps
	allocator      *allocator.Allocator
	globalHashrate *hashrate.GlobalHashrate
	log            interfaces.ILogger
}

func NewContractWatcherBuyer(
	terms *hashrateContract.EncryptedTerms,
	hashrateFactory func() *hashrate.Hashrate,
	allocator *allocator.Allocator,
	globalHashrate *hashrate.GlobalHashrate,
	log interfaces.ILogger,

	cycleDuration time.Duration,
	validationStartTimeout time.Duration,
	shareTimeout time.Duration,
	hrErrorThreshold float64,
) *ContractWatcherBuyer {
	return &ContractWatcherBuyer{
		terms:          terms,
		state:          resources.ContractStatePending,
		allocator:      allocator,
		globalHashrate: globalHashrate,
		log:            log,

		contractCycleDuration:  cycleDuration,
		validationStartTimeout: validationStartTimeout,
		shareTimeout:           shareTimeout,
		hrErrorThreshold:       hrErrorThreshold,
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

	endTimer := time.NewTimer(time.Until(*p.GetEndTime()))

	for {
		err := p.checkIncomingHashrate(ctx)
		if err != nil {
			return err
		}

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
		}
	}
}

func (p *ContractWatcherBuyer) proceedToNextStage() {
	if p.validationStage == ValidationStageNotValidating && p.isValidationStartTimeout() {
		p.validationStage = ValidationStageValidating
		p.log.Infof("new validation stage %s", p.validationStage)
		return
	}

	if p.isContractExpired() {
		p.validationStage = ValidationStageFinished
		p.log.Infof("new validation stage %s", p.validationStage)
		return
	}
}

func (p *ContractWatcherBuyer) checkIncomingHashrate(ctx context.Context) error {
	p.proceedToNextStage()

	isHashrateOK := p.isReceivingAcceptableHashrate()

	switch p.validationStage {
	case ValidationStageNotValidating:
		lastShareTime, ok := p.globalHashrate.GetLastSubmitTime(p.getWorkerName())
		if !ok {
			lastShareTime = *p.fulfillmentStartedAt
		}
		if time.Since(lastShareTime) > p.shareTimeout {
			return fmt.Errorf("no share submitted within shareTimeout (%s)", p.shareTimeout)
		}
		return nil
	case ValidationStageValidating:
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
	case ValidationStageFinished:
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
		"worker %s, target GHS %.0f, actual GHS %.0f, error %.0f%%, threshold(%.0f%%)",
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
