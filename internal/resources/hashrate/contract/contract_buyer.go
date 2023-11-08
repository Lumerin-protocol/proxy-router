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
	contractCycleDuration    time.Duration
	shareTimeout             time.Duration // time to wait for the share to arrive, otherwise close contract
	hrErrorThreshold         float64       // hashrate relative error threshold for the contract to be considered fulfilling accurately
	hashrateCounterNameBuyer string

	*hashrateContract.EncryptedTerms
	state                  resources.ContractState
	validationStage        hashrateContract.ValidationStage
	fulfillmentStartedAt   time.Time
	validatorGraceDuration time.Duration

	tsk    *lib.Task
	cancel context.CancelFunc
	err    error
	doneCh chan struct{}

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
	shareTimeout time.Duration,
	hrErrorThreshold float64,
	hashrateCounterNameBuyer string,
	validatorGraceDuration time.Duration,
) *ContractWatcherBuyer {
	return &ContractWatcherBuyer{
		EncryptedTerms:           terms,
		state:                    resources.ContractStatePending,
		allocator:                allocator,
		globalHashrate:           globalHashrate,
		log:                      log,
		contractCycleDuration:    cycleDuration,
		shareTimeout:             shareTimeout,
		hrErrorThreshold:         hrErrorThreshold,
		validationStage:          hashrateContract.ValidationStageValidating,
		hashrateCounterNameBuyer: hashrateCounterNameBuyer,
		validatorGraceDuration:   validatorGraceDuration,
	}
}

func (p *ContractWatcherBuyer) StartFulfilling(ctx context.Context) {
	if p.state == resources.ContractStateRunning {
		p.log.Infof("buyer contract already fulfilling")
		return
	}
	p.log.Infof("buyer contract started fulfilling")
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.doneCh = make(chan struct{})

	go func() {
		p.state = resources.ContractStateRunning
		p.err = p.Run(ctx)
		close(p.doneCh)
		p.state = resources.ContractStatePending
	}()
}

func (p *ContractWatcherBuyer) StopFulfilling() {
	p.cancel()
	<-p.doneCh
	p.log.Infof("buyer contract stopped fulfilling")
}

func (p *ContractWatcherBuyer) Done() <-chan struct{} {
	return p.doneCh
}

func (p *ContractWatcherBuyer) Err() error {
	if errors.Is(p.err, context.Canceled) {
		return ErrContractClosed
	}
	return p.err
}

func (p *ContractWatcherBuyer) SetData(terms *hashrateContract.EncryptedTerms) {
	p.EncryptedTerms = terms
}

func (p *ContractWatcherBuyer) Run(ctx context.Context) error {
	p.state = resources.ContractStateRunning
	startedAt := time.Now()
	p.fulfillmentStartedAt = startedAt

	err := p.fillBuyerCounterGraceData()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(p.contractCycleDuration)
	defer ticker.Stop()

	tillEndTime := time.Until(p.EndTime())
	if tillEndTime <= 0 {
		return nil
	}

	endTimer := time.NewTimer(tillEndTime)

	for {
		err := p.checkIncomingHashrate(ctx)
		if err != nil {
			return err
		}

		tillEndTime := time.Until(p.EndTime())
		if tillEndTime <= 0 {
			return nil
		}
		endTimer.Reset(tillEndTime)

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

func (p *ContractWatcherBuyer) fillBuyerCounterGraceData() error {
	p.globalHashrate.Reset(p.ID())
	p.globalHashrate.Initialize(p.ID())
	worker := p.globalHashrate.GetWorker(p.getWorkerName())
	if worker == nil {
		return fmt.Errorf("failed to get worker")
	}

	counter := worker.GetHashrateCounter(p.hashrateCounterNameBuyer)
	if counter == nil {
		return fmt.Errorf("missing sma hashrate counter")
	}

	sma, ok := counter.(*hashrate.Sma)
	if !ok {
		return fmt.Errorf("failed to get sma")
	}

	shareTime := 15 * time.Second

	totalJob := hashrate.GHSToJobSubmitted(p.HashrateGHS()) * p.validatorGraceDuration.Seconds()
	totalShares := int(p.validatorGraceDuration / shareTime)
	jobPerShare := totalJob / float64(totalShares)
	remainder := totalJob - jobPerShare*float64(totalShares)
	for i := totalShares - 1; i > 0; i-- {
		sma.AddWithTimestamp(jobPerShare, p.fulfillmentStartedAt.Add(-time.Duration(i)*shareTime))
	}
	// add remainder to the last share
	sma.AddWithTimestamp(jobPerShare+remainder, p.fulfillmentStartedAt)

	return nil
}

func (p *ContractWatcherBuyer) proceedToNextStage() {
	if p.isContractExpired() {
		p.validationStage = hashrateContract.ValidationStageFinished
		p.log.Infof("new validation stage %s", p.validationStage.String())
		return
	}
}

func (p *ContractWatcherBuyer) checkIncomingHashrate(ctx context.Context) error {
	p.proceedToNextStage()

	isHashrateOK := p.isReceivingAcceptableHashrate()

	switch p.validationStage {
	case hashrateContract.ValidationStageValidating:
		lastShareTime, ok := p.globalHashrate.GetLastSubmitTime(p.getWorkerName())
		if !ok {
			lastShareTime = p.fulfillmentStartedAt
		}

		if time.Since(lastShareTime) > p.shareTimeout {
			return fmt.Errorf("no share submitted within shareTimeout (%s), lastShare at (%s)", p.shareTimeout, lastShareTime.Format(time.RFC3339))
		}
		if !isHashrateOK {
			return fmt.Errorf("contract is not delivering accurate hashrate")
		}
		return nil
	case hashrateContract.ValidationStageFinished:
		return fmt.Errorf("contract is finished")
	default:
		return fmt.Errorf("unknown validation state")
	}
}

func (p *ContractWatcherBuyer) isReceivingAcceptableHashrate() bool {
	actualHashrate, ok := p.globalHashrate.GetHashRateGHS(p.getWorkerName(), p.hashrateCounterNameBuyer)
	if !ok {
		p.log.Warnf("no hashrate submitted yet")
	}
	targetHashrateGHS := p.HashrateGHS()

	hrError := lib.RelativeError(targetHashrateGHS, actualHashrate)

	hrMsg := fmt.Sprintf(
		"elapsed %s worker %s, target GHS %.0f, actual GHS %.0f, error %.0f%%, threshold(%.0f%%)",
		p.Elapsed().Round(time.Second), p.getWorkerName(), targetHashrateGHS, actualHashrate, hrError*100, p.hrErrorThreshold*100,
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

func (p *ContractWatcherBuyer) isContractExpired() bool {
	return time.Now().After(p.EndTime())
}

func (p *ContractWatcherBuyer) getWorkerName() string {
	return p.ID()
}

func (p *ContractWatcherBuyer) Role() resources.ContractRole {
	return resources.ContractRoleBuyer
}

func (p *ContractWatcherBuyer) FulfillmentStartTime() time.Time {
	return p.fulfillmentStartedAt
}

func (p *ContractWatcherBuyer) State() resources.ContractState {
	return p.state
}

func (p *ContractWatcherBuyer) ValidationStage() hashrateContract.ValidationStage {
	return p.validationStage
}

func (p *ContractWatcherBuyer) ResourceEstimates() map[string]float64 {
	return map[string]float64{
		ResourceEstimateHashrateGHS: p.HashrateGHS(),
	}
}

func (p *ContractWatcherBuyer) ResourceEstimatesActual() map[string]float64 {
	res, _ := p.globalHashrate.GetHashRateGHSAll(p.getWorkerName())
	return res
}

func (p *ContractWatcherBuyer) ResourceType() string {
	return ResourceTypeHashrate
}

func (p *ContractWatcherBuyer) Dest() string {
	// the destination is localhost for the buyer
	return ""
}
