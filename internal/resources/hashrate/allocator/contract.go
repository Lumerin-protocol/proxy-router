package allocator

import (
	"context"
	"net/url"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"golang.org/x/exp/slices"
)

const (
	ContractCycleDuration = 5 * time.Minute
)

type ContractWatcher struct {
	contractID  string
	hashrateGHS float64
	dest        *url.URL
	startedAt   time.Time
	duration    time.Duration

	fullMiners  []string
	actualHRGHS hashrate.Hashrate

	//deps
	allocator *Allocator
	log       *lib.Logger
}

func NewContractWatcher(contractID string, hashrateGHS float64, dest *url.URL, startedAt time.Time, duration time.Duration, allocator *Allocator) *ContractWatcher {
	return &ContractWatcher{
		contractID:  contractID,
		hashrateGHS: hashrateGHS,
		dest:        dest,
		startedAt:   startedAt,
		duration:    duration,

		allocator:   allocator,
		fullMiners:  []string{},
		actualHRGHS: *hashrate.NewHashrate(),
	}
}

func (p *ContractWatcher) Run(ctx context.Context) error {
	remainderGHS := p.hashrateGHS

	for {
		if remainderGHS > 0 {
			fullMiners, newRemainderGHS := p.allocator.AllocateFullMinersForHR(remainderGHS, p.dest, p.duration, p.actualHRGHS.OnSubmit)
			if len(fullMiners) > 0 {
				remainderGHS = newRemainderGHS
				p.fullMiners = append(p.fullMiners, fullMiners...)
			}
			p.allocator.AllocatePartialMinersForHROneTime(remainderGHS, p.dest, ContractCycleDuration, p.actualHRGHS.OnSubmit)
		}

		if remainderGHS < 0 {
			p.log.Info("removing least powerful miner from contract")
			var items []*MinerItem
			for _, minerID := range p.fullMiners {
				miner, ok := p.allocator.proxies.Load(minerID)
				if !ok {
					continue
				}
				items = append(items, &MinerItem{
					ID:    miner.GetID(),
					HrGHS: miner.HashrateGHS(),
				})
			}

			if len(items) > 0 {
				slices.SortStableFunc(items, func(a, b *MinerItem) bool {
					return b.HrGHS > a.HrGHS
				})
				minerToRemove := items[0].ID
				miner, ok := p.allocator.proxies.Load(minerToRemove)
				if ok {
					// replace with more specific remove (by tag which could be a contractID)
					miner.RemoveTasks()
				}
			} else {
				p.log.Warnf("no miners found")
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Until(p.startedAt.Add(p.duration))):
			return nil
		case <-time.After(ContractCycleDuration):
		}

		lastCycleGHS := 0.0 // TODO: get job submitted for last cycle
		underSubmittedGHS := p.hashrateGHS - lastCycleGHS
		remainderGHS += underSubmittedGHS
	}
}
