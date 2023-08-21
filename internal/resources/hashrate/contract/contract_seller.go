package contract

import (
	"context"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/contractmanager"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"golang.org/x/exp/slices"
)

const (
	ContractCycleDuration = 5 * time.Minute
)

type ContractWatcher struct {
	data *contractmanager.ContractData

	state                contractmanager.ContractState
	fullMiners           []string
	actualHRGHS          hashrate.Hashrate
	fulfillmentStartedAt *time.Time

	//deps
	allocator *allocator.Allocator
	log       interfaces.ILogger
}

const (
	ResourceTypeHashrate        = "hashrate"
	ResourceEstimateHashrateGHS = "hashrate_ghs"
)

func NewContractWatcherSeller(data *contractmanager.ContractData, allocator *allocator.Allocator, log interfaces.ILogger) *ContractWatcher {
	return &ContractWatcher{
		data:        data,
		state:       contractmanager.ContractStatePending,
		allocator:   allocator,
		fullMiners:  []string{},
		actualHRGHS: *hashrate.NewHashrate(),
		log:         log,
	}
}

func (p *ContractWatcher) Run(ctx context.Context) error {
	remainderGHS := p.GetHashrateGHS()
	lastCycleJobSubmitted := 0.0

	onSubmit := func(diff float64) {
		p.actualHRGHS.OnSubmit(diff)
		lastCycleJobSubmitted += diff
	}

	for {
		p.log.Debugf("new contract cycle: remainderGHS=%.1f", remainderGHS)
		if remainderGHS > 0 {
			fullMiners, newRemainderGHS := p.allocator.AllocateFullMinersForHR(remainderGHS, p.data.Dest, p.GetDuration(), onSubmit)
			if len(fullMiners) > 0 {
				p.log.Debugf("allocated full miners: %v", fullMiners)
				remainderGHS = newRemainderGHS
				p.fullMiners = append(p.fullMiners, fullMiners...)
			} else {
				p.log.Debugf("no full miners were allocated for this contract")
			}

			minerID, ok := p.allocator.AllocatePartialForHR(remainderGHS, p.data.Dest, ContractCycleDuration, onSubmit)
			if ok {
				p.log.Debugf("remainderGHS: %.1f, was allocated by partial miners %v", remainderGHS, minerID)
			} else {
				p.log.Warnf("remainderGHS: %.1f, was not allocated by partial miners", remainderGHS)
			}
		}

		// in case of too much hashrate
		if remainderGHS < 0 {
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
					miner.RemoveTasks()
					p.log.Debugf("miner %s tasks removed", miner.GetID())
				}
			} else {
				p.log.Warnf("no miners found to be removed")
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Until(*p.GetEndTime())):
			p.log.Warnf("contract ended")
			return nil
		case <-time.After(ContractCycleDuration):
		}

		lastCycleGHS := hashrate.JobSubmittedToGHS(lastCycleJobSubmitted / ContractCycleDuration.Seconds())
		lastCycleJobSubmitted = 0.0
		underSubmittedGHS := p.GetHashrateGHS() - lastCycleGHS
		remainderGHS += underSubmittedGHS

		p.log.Infof(
			"contract cycle ended, lastCycleGHS=%.1f, underSubmittedGHS=%.1f, remainderGHS=%.1f",
			lastCycleGHS, underSubmittedGHS, remainderGHS,
		)
	}
}

func (p *ContractWatcher) GetRole() contractmanager.ContractRole {
	return contractmanager.ContractRoleSeller
}

func (p *ContractWatcher) GetDest() string {
	return p.data.Dest.String()
}

func (p *ContractWatcher) GetDuration() time.Duration {
	return p.data.Duration
}

func (p *ContractWatcher) GetEndTime() *time.Time {
	if p.data.StartedAt == nil {
		return nil
	}
	endTime := p.data.StartedAt.Add(p.data.Duration)
	return &endTime
}

func (p *ContractWatcher) GetFulfillmentStartedAt() *time.Time {
	return p.fulfillmentStartedAt
}

func (p *ContractWatcher) GetID() string {
	return p.data.ContractID
}

func (p *ContractWatcher) GetHashrateGHS() float64 {
	return p.data.ResourceEstimates[ResourceEstimateHashrateGHS]
}

func (p *ContractWatcher) GetResourceEstimates() map[string]float64 {
	return p.data.ResourceEstimates
}

func (p *ContractWatcher) GetResourceEstimatesActual() map[string]float64 {
	return map[string]float64{
		ResourceEstimateHashrateGHS: float64(p.actualHRGHS.GetHashrateGHS()),
	}
}

func (p *ContractWatcher) GetResourceType() string {
	return ResourceTypeHashrate
}

func (p *ContractWatcher) GetSeller() string {
	return p.data.Seller
}

func (p *ContractWatcher) GetBuyer() string {
	return p.data.Buyer
}

func (p *ContractWatcher) GetStartedAt() *time.Time {
	return p.data.StartedAt
}

func (p *ContractWatcher) GetState() contractmanager.ContractState {
	return p.state
}
