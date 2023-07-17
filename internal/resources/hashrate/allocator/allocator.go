package allocator

import (
	"context"
	"net/url"
	"sync"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"golang.org/x/exp/slices"
)

type MinerItem struct {
	ID    string
	HrGHS float64
}

type MinerItemJobScheduled struct {
	ID       string
	Job      float64
	Fraction float64
}

type AllocatorInterface interface {
	Run(ctx context.Context) error
	UpsertAllocation(ID string, hashrate float64, dest string, counter func(diff float64)) error
}

type Allocator struct {
	proxies    *lib.Collection[*Scheduler]
	proxyState sync.Map // map[string]bool map[proxyID]contractID
}

func NewAllocator(proxies *lib.Collection[*Scheduler]) *Allocator {
	return &Allocator{
		proxies:    proxies,
		proxyState: sync.Map{},
	}
}

func (p *Allocator) Run(ctx context.Context) error {
	return nil
}

func (p *Allocator) AllocateFullMinersForHR(hrGHS float64, dest *url.URL, duration time.Duration, onSubmit func(diff float64)) (minerIDs []string, deltaGHS float64) {
	miners := p.GetFreeMiners()

	for _, miner := range miners {
		if miner.HrGHS <= hrGHS {
			minerIDs = append(minerIDs, miner.ID)
			proxy, ok := p.proxies.Load(miner.ID)
			if ok {
				proxy.AddTask(dest, hashrate.GHSToJobSubmitted(miner.HrGHS)*duration.Seconds(), onSubmit)
			}
			hrGHS -= miner.HrGHS
		}
	}

	return minerIDs, hrGHS

	// TODO: improve miner selection
	// sort.Slice(miners, func(i, j int) bool {
	// 	return miners[i].HrGHS > miners[j].HrGHS
	// })
}

func (p *Allocator) AllocatePartialMinersForHROneTime(hrGHS float64, dest *url.URL, cycleDuration time.Duration, onSubmit func(diff float64)) (delta float64) {
	miners := p.GetPartialMiners()
	// minerIDs := []string{}
	jobForCycle := hashrate.GHSToJobSubmitted(hrGHS) * cycleDuration.Seconds()

	// search in partially allocated miners
	for _, miner := range miners {
		remainingJob := miner.Job / miner.Fraction
		if remainingJob >= jobForCycle {
			m, ok := p.proxies.Load(miner.ID)
			if ok {
				m.AddTask(dest, jobForCycle, onSubmit)
				return
			}
		}
	}

	// search in free miners
	for _, miner := range p.GetFreeMiners() {
		remainingJob := hashrate.GHSToJobSubmitted(miner.HrGHS)
		if remainingJob >= jobForCycle {
			m, ok := p.proxies.Load(miner.ID)
			if ok {
				m.AddTask(dest, jobForCycle, onSubmit)
				return
			}
		}
	}

	return 0
}

func (p *Allocator) GetFreeMiners() []MinerItem {
	freeMiners := []MinerItem{}
	p.proxies.Range(func(item *Scheduler) bool {
		if item.IsFree() {
			freeMiners = append(freeMiners, MinerItem{
				ID:    item.GetID(),
				HrGHS: item.HashrateGHS(),
			})
		}
		return true
	})

	slices.SortStableFunc(freeMiners, func(i, j MinerItem) bool {
		return i.HrGHS > j.HrGHS
	})

	return freeMiners
}

func (p *Allocator) GetPartialMiners() []MinerItemJobScheduled {
	partialMiners := []MinerItemJobScheduled{}
	p.proxies.Range(func(item *Scheduler) bool {
		if item.IsAcceptingTasks(ContractCycleDuration) {
			partialMiners = append(partialMiners, MinerItemJobScheduled{
				ID:       item.GetID(),
				Job:      item.GetTotalTaskJob(),
				Fraction: float64(hashrate.JobSubmittedToGHS(item.GetTotalTaskJob())) / item.HashrateGHS(),
			})
		}
		return true
	})

	slices.SortStableFunc(partialMiners, func(i, j MinerItemJobScheduled) bool {
		return i.Fraction > j.Fraction
	})

	return partialMiners
}

func (p *Allocator) GetMiners() *lib.Collection[*Scheduler] {
	return p.proxies
}
