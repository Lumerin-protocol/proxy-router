package contract

import (
	"context"
	"errors"
	"net/url"
	"sync/atomic"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate"
	hashrateContract "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	hr "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"golang.org/x/exp/slices"
)

type ContractWatcherSeller struct {
	terms *hashrateContract.Terms

	state                 resources.ContractState
	fullMiners            []string
	actualHRGHS           *hr.Hashrate
	fulfillmentStartedAt  *time.Time
	contractCycleDuration time.Duration

	tsk *lib.Task

	//deps
	allocator *allocator.Allocator
	log       interfaces.ILogger
}

func NewContractWatcherSeller(data *hashrateContract.Terms, cycleDuration time.Duration, hashrateFactory func() *hr.Hashrate, allocator *allocator.Allocator, log interfaces.ILogger) *ContractWatcherSeller {
	p := &ContractWatcherSeller{
		terms:                 data,
		state:                 resources.ContractStatePending,
		allocator:             allocator,
		fullMiners:            []string{},
		contractCycleDuration: cycleDuration,
		actualHRGHS:           hashrateFactory(),
		log:                   log,
	}
	p.tsk = lib.NewTaskFunc(p.Run)
	return p
}

func (p *ContractWatcherSeller) StartFulfilling(ctx context.Context) {
	p.log.Infof("contract started fulfilling")
	startedAt := time.Now()
	p.fulfillmentStartedAt = &startedAt
	p.state = resources.ContractStateRunning
	p.tsk.Start(ctx)
}

func (p *ContractWatcherSeller) StopFulfilling() {
	<-p.tsk.Stop()
	p.allocator.CancelTasks(p.GetID())
	p.state = resources.ContractStatePending
	p.log.Infof("contract stopped fulfilling")
}

func (p *ContractWatcherSeller) Done() <-chan struct{} {
	return p.tsk.Done()
}

func (p *ContractWatcherSeller) Err() error {
	if errors.Is(p.tsk.Err(), context.Canceled) {
		return ErrContractClosed
	}
	return p.tsk.Err()
}

func (p *ContractWatcherSeller) SetData(data *hashrateContract.Terms) {
	p.terms = data
}

// Run is the main loop of the contract. It is responsible for allocating miners for the contract.
// Returns nil if the contract ended successfully, ErrClosed if the contract was closed before it ended.
func (p *ContractWatcherSeller) Run(ctx context.Context) error {
	partialDeliveryTargetGHS := p.GetHashrateGHS()
	thisCycleJobSubmitted := atomic.Uint64{}
	thisCyclePartialAllocation := 0.0

	onSubmit := func(diff float64, minerID string) {
		p.log.Infof("contract submit %s, %.0f, total work %d", minerID, diff, thisCycleJobSubmitted.Load())
		p.actualHRGHS.OnSubmit(diff)
		thisCycleJobSubmitted.Add(uint64(diff))
		// TODO: catch overdelivery here and cancel tasks
	}

	for {
		p.log.Debugf("new contract cycle:  partialDeliveryTargetGHS=%.0f, thisCyclePartialAllocation=%.0f",
			partialDeliveryTargetGHS, thisCyclePartialAllocation,
		)
		if partialDeliveryTargetGHS > 0 {
			fullMiners, newRemainderGHS := p.allocator.AllocateFullMinersForHR(p.terms.GetID(), partialDeliveryTargetGHS, p.getAdjustedDest(), p.GetDuration(), onSubmit)
			if len(fullMiners) > 0 {
				partialDeliveryTargetGHS = newRemainderGHS
				p.log.Infof("fully allocated %d miners, new partialDeliveryTargetGHS = %.0f", len(fullMiners), partialDeliveryTargetGHS)
				p.fullMiners = append(p.fullMiners, fullMiners...)
			} else {
				p.log.Debugf("no full miners were allocated for this contract")
			}

			thisCyclePartialAllocation = partialDeliveryTargetGHS
			minerID, ok := p.allocator.AllocatePartialForHR(p.GetID(), partialDeliveryTargetGHS, p.getAdjustedDest(), p.contractCycleDuration, onSubmit)
			if ok {
				p.log.Debugf("remainderGHS: %.0f, was allocated by partial miners %v", partialDeliveryTargetGHS, minerID)
			} else {
				p.log.Warnf("remainderGHS: %.0f, was not allocated by partial miners", partialDeliveryTargetGHS)
			}
		}

		// in case of too much hashrate
		if partialDeliveryTargetGHS < 0 {
			p.log.Info("removing least powerful miner from contract")
			var items []*allocator.MinerItem
			for _, minerID := range p.fullMiners {
				miner, ok := p.allocator.GetMiners().Load(minerID)
				if !ok {
					continue
				}
				items = append(items, &allocator.MinerItem{
					ID:    miner.GetID(),
					HrGHS: miner.HashrateGHS(),
				})
			}

			if len(items) > 0 {
				slices.SortStableFunc(items, func(a, b *allocator.MinerItem) bool {
					return b.HrGHS > a.HrGHS
				})
				minerToRemove := items[0].ID
				miner, ok := p.allocator.GetMiners().Load(minerToRemove)
				if ok {
					// replace with more specific remove (by tag which could be a contractID)
					miner.ResetTasks()
					p.log.Debugf("miner %s tasks removed", miner.GetID())
					// TODO: remove from full miners
					newFullMiners := make([]string, len(p.fullMiners)-1)
					i := 0
					for _, minerID := range p.fullMiners {
						if minerID == minerToRemove {
							continue
						}
						newFullMiners[i] = minerID
						i++
					}
					p.fullMiners = newFullMiners

					// sets new target and restarts the cycle
					partialDeliveryTargetGHS = miner.HashrateGHS() + partialDeliveryTargetGHS
					continue
				}
			} else {
				p.log.Warnf("no miners found to be removed")
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Until(*p.GetEndTime())):
			expectedJob := hr.GHSToJobSubmitted(p.GetHashrateGHS()) * p.GetDuration().Seconds()
			actualJob := p.actualHRGHS.GetTotalWork()
			undeliveredJob := expectedJob - actualJob
			undeliveredFraction := undeliveredJob / expectedJob

			for _, minerID := range p.fullMiners {
				miner, ok := p.allocator.GetMiners().Load(minerID)
				if !ok {
					continue
				}
				miner.ResetTasks()
				p.log.Debugf("miner %s tasks removed", miner.GetID())
			}
			p.fullMiners = p.fullMiners[:0]

			// partial miners tasks are not reset because they are not allocated
			// for the full duration of the contract

			p.log.Infof("contract ended, undelivered work %d, undelivered fraction %.2f",
				int(undeliveredJob), undeliveredFraction)
			return nil
		case <-time.After(p.contractCycleDuration):
		}

		thisCycleActualGHS := hr.JobSubmittedToGHS(float64(thisCycleJobSubmitted.Load()) / p.contractCycleDuration.Seconds())
		thisCycleUnderDeliveryGHS := p.GetHashrateGHS() - thisCycleActualGHS

		// plan for the next cycle is to compensate for the under delivery of this cycle
		partialDeliveryTargetGHS = partialDeliveryTargetGHS + thisCycleUnderDeliveryGHS

		thisCycleJobSubmitted.Store(0)

		p.log.Infof(
			"contract cycle ended, thisCycleActualGHS = %.0f, thisCycleUnderDeliveryGHS=%.0f, partialDeliveryTargetGHS=%.0f",
			thisCycleActualGHS, thisCycleUnderDeliveryGHS, partialDeliveryTargetGHS,
		)
	}
}

// getAdjustedDest returns the destination url with the username set to the contractID
// this is required for the buyer to distinguish incoming hashrate between different contracts
func (p *ContractWatcherSeller) getAdjustedDest() *url.URL {
	if p.terms.Dest == nil {
		return nil
	}
	dest := *p.terms.Dest
	lib.SetUserName(&dest, p.terms.ContractID)
	return &dest
}

// ShouldBeRunning checks blockchain state and expiration time and returns true if the contract should be running
func (p *ContractWatcherSeller) ShouldBeRunning() bool {
	endTime := p.GetEndTime()
	if endTime == nil {
		return false
	}
	return p.GetBlockchainState() == hashrate.BlockchainStateRunning && p.GetEndTime().After(time.Now())
}

//
// Public getters
//

func (p *ContractWatcherSeller) GetRole() resources.ContractRole {
	return resources.ContractRoleSeller
}

func (p *ContractWatcherSeller) GetDest() string {
	if dest := p.getAdjustedDest(); dest != nil {
		return dest.String()
	}
	return ""
}

func (p *ContractWatcherSeller) GetDuration() time.Duration {
	return p.terms.Duration
}

func (p *ContractWatcherSeller) GetStartedAt() *time.Time {
	return p.terms.StartsAt
}

func (p *ContractWatcherSeller) GetEndTime() *time.Time {
	if p.terms.StartsAt == nil {
		return nil
	}
	endTime := p.terms.StartsAt.Add(p.terms.Duration)
	return &endTime
}

func (p *ContractWatcherSeller) GetFulfillmentStartedAt() *time.Time {
	return p.fulfillmentStartedAt
}

func (p *ContractWatcherSeller) GetElapsed() *time.Duration {
	if p.terms.StartsAt == nil {
		return nil
	}
	res := time.Since(*p.terms.StartsAt)
	return &res
}

func (p *ContractWatcherSeller) GetID() string {
	return p.terms.ContractID
}

func (p *ContractWatcherSeller) GetHashrateGHS() float64 {
	return p.terms.Hashrate
}

func (p *ContractWatcherSeller) GetSeller() string {
	return p.terms.Seller
}

func (p *ContractWatcherSeller) GetBuyer() string {
	return p.terms.Buyer
}

func (p *ContractWatcherSeller) GetState() resources.ContractState {
	return p.state
}

func (p *ContractWatcherSeller) GetBlockchainState() hashrate.BlockchainState {
	return p.terms.State
}

func (p *ContractWatcherSeller) GetResourceType() string {
	return ResourceTypeHashrate
}

func (p *ContractWatcherSeller) GetResourceEstimates() map[string]float64 {
	return map[string]float64{
		ResourceEstimateHashrateGHS: p.GetHashrateGHS(),
	}
}

func (p *ContractWatcherSeller) GetResourceEstimatesActual() map[string]float64 {
	return p.actualHRGHS.GetHashrateAvgGHSAll()
}

func (p *ContractWatcherSeller) GetValidationStage() hashrateContract.ValidationStage {
	return hashrateContract.ValidationStageNotApplicable // only for buyer
}
