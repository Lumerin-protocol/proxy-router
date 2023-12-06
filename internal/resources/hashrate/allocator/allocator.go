package allocator

import (
	"context"
	"math"
	"net/url"
	"sort"
	"sync"
	"time"

	gi "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"golang.org/x/exp/slices"
)

const (
	HashratePredictionAdjustment = 1.0
)

type MinerItem struct {
	ID           string
	HrGHS        float64
	JobRemaining float64
	IsFullMiner  bool
}

type ListenerHandle int

type MinerItemJobScheduled struct {
	ID       string
	Job      float64
	Fraction float64
}

type PartialMinerJobRemaining struct {
	ID           string
	JobRemaining float64
}

type MinerIDJob = map[string]float64

type AllocatorInterface interface {
	Run(ctx context.Context) error
	UpsertAllocation(ID string, hashrate float64, dest string, counter func(diff float64)) error
}

type Allocator struct {
	proxies    *lib.Collection[*Scheduler]
	proxyMutex sync.RWMutex

	lastListenerID  int
	vettedListeners map[int]func(ID string)
	vettedMutex     sync.RWMutex

	log gi.ILogger
}

func NewAllocator(proxies *lib.Collection[*Scheduler], log gi.ILogger) *Allocator {
	return &Allocator{
		proxies:         proxies,
		vettedListeners: make(map[int]func(ID string), 0),
		log:             log,
	}
}

func (p *Allocator) AllocateFullMinersForHR(
	ID string,
	hrGHS float64,
	dest *url.URL,
	duration time.Duration,
	onSubmit OnSubmitCb,
	onDisconnect OnDisconnectCb,
	onEnd OnEndCb,
) (minerIDs []string, deltaGHS float64) {
	p.proxyMutex.Lock()
	defer p.proxyMutex.Unlock()

	miners := p.getFreeMiners()
	p.log.Infof("available free miners %v", miners)

	sort.Slice(miners, func(i, j int) bool {
		return miners[i].HrGHS < miners[j].HrGHS
	})

	for _, miner := range miners {
		minerGHS := miner.HrGHS
		if minerGHS <= hrGHS && minerGHS > 0 {
			proxy, ok := p.proxies.Load(miner.ID)
			if ok {
				proxy.AddTask(ID, dest, hashrate.GHSToJobSubmittedV2(minerGHS, duration), onSubmit, onDisconnect, onEnd, time.Now().Add(duration))
				minerIDs = append(minerIDs, miner.ID)
				hrGHS -= minerGHS
				p.log.Infof("full miner %s allocated for %.0f GHS", miner.ID, minerGHS)
			}
		}
	}

	return minerIDs, hrGHS
}

func (p *Allocator) AllocatePartialForJob(
	ID string,
	jobNeeded float64,
	dest *url.URL,
	cycleEndTimeout time.Duration,
	onSubmit func(diff float64, ID string),
	onDisconnect func(ID string, hrGHS float64, remainingJob float64),
	onEnd OnEndCb,
) (minerIDJob MinerIDJob, remainderGHS float64) {
	p.proxyMutex.Lock()
	defer p.proxyMutex.Unlock()

	p.log.Infof("attempting to partially allocate job %.f", jobNeeded)

	partialMiners := p.getPartialMiners(cycleEndTimeout)
	p.log.Infof("available partial miners %v", partialMiners)
	minerIDJob = MinerIDJob{}

	minJob := 5000.0

	for _, miner := range partialMiners {
		if jobNeeded <= minJob {
			return minerIDJob, 0
		}

		// try to add the whole chunk and return
		if miner.JobRemaining >= jobNeeded {
			m, ok := p.proxies.Load(miner.ID)
			if ok {
				m.AddTask(ID, dest, jobNeeded, onSubmit, onDisconnect, onEnd, time.Now().Add(cycleEndTimeout))
				minerIDJob[miner.ID] = jobNeeded
				return minerIDJob, 0
			}
		}

		// try to add at least a minJob and continue
		if miner.JobRemaining >= minJob {
			m, ok := p.proxies.Load(miner.ID)
			if ok {
				m.AddTask(ID, dest, miner.JobRemaining, onSubmit, onDisconnect, onEnd, time.Now().Add(cycleEndTimeout))
				minerIDJob[miner.ID] = miner.JobRemaining
				jobNeeded -= miner.JobRemaining
			}
		}
	}

	// search in free miners
	// missing loop cause we already checked full miners
	freeMiners := p.getFreeMiners()
	p.log.Infof("available free miners %v", freeMiners)

	for _, miner := range freeMiners {
		if jobNeeded < minJob {
			jobNeeded = 0
			break
		}

		minerJobRemaining := hashrate.GHSToJobSubmittedV2(miner.HrGHS, cycleEndTimeout)
		if minerJobRemaining <= minJob {
			continue
		}

		jobToAllocate := math.Min(minerJobRemaining, jobNeeded)

		m, ok := p.proxies.Load(miner.ID)
		if !ok {
			continue
		}

		m.AddTask(ID, dest, jobToAllocate, onSubmit, onDisconnect, onEnd, time.Now().Add(cycleEndTimeout))
		minerIDJob[miner.ID] = jobToAllocate
		jobNeeded -= jobToAllocate
	}

	return minerIDJob, jobNeeded
}

func (p *Allocator) getFreeMiners() []MinerItem {
	freeMiners := []MinerItem{}
	p.proxies.Range(func(item *Scheduler) bool {
		if item.IsVetting() {
			return true
		}
		if item.IsDisconnecting() {
			return true
		}
		if item.IsFree() {
			freeMiners = append(freeMiners, MinerItem{
				ID:    item.ID(),
				HrGHS: item.HashrateGHS() * HashratePredictionAdjustment,
			})
		}
		return true
	})

	slices.SortStableFunc(freeMiners, func(i, j MinerItem) bool {
		return i.HrGHS > j.HrGHS
	})

	return freeMiners
}

func (p *Allocator) getPartialMiners(remainingCycleDuration time.Duration) []PartialMinerJobRemaining {
	partialMiners := []PartialMinerJobRemaining{}

	p.proxies.Range(func(item *Scheduler) bool {
		if item.IsVetting() {
			return true
		}
		if item.IsDisconnecting() {
			return true
		}

		if item.IsPartialBusy(remainingCycleDuration) {
			partialMiners = append(partialMiners, PartialMinerJobRemaining{
				ID:           item.ID(),
				JobRemaining: item.GetJobCouldBeScheduledTill(remainingCycleDuration),
			})
		}

		return true
	})

	slices.SortStableFunc(partialMiners, func(i, j PartialMinerJobRemaining) bool {
		return i.JobRemaining < j.JobRemaining
	})

	return partialMiners
}

func (p *Allocator) GetMiners() *lib.Collection[*Scheduler] {
	p.proxyMutex.RLock()
	defer p.proxyMutex.RUnlock()

	return p.proxies
}

func (p *Allocator) GetMinersFulfillingContract(contractID string, cycleDuration time.Duration) []*MinerItemJobScheduled {
	p.proxyMutex.RLock()
	defer p.proxyMutex.RUnlock()

	minerItems := []*MinerItemJobScheduled{}

	p.GetMiners().Range(func(item *Scheduler) bool {
		if item.IsVetting() {
			return true
		}

		if item.IsDisconnecting() {
			return true
		}

		tasks := item.GetTasksByID(contractID)
		maxJob := item.getExpectedCycleJob(cycleDuration)

		for _, task := range tasks {
			job := float64(task.RemainingJobToSubmit.Load())
			minerItems = append(minerItems, &MinerItemJobScheduled{
				ID:       item.ID(),
				Job:      job,
				Fraction: job / maxJob,
			})
		}
		return true
	})

	return minerItems
}

func (p *Allocator) AddVettedListener(f func(ID string)) ListenerHandle {
	p.vettedMutex.Lock()
	defer p.vettedMutex.Unlock()

	ID := p.lastListenerID
	p.lastListenerID++
	p.vettedListeners[ID] = f

	return ListenerHandle(ID)
}

func (p *Allocator) RemoveVettedListener(s ListenerHandle) {
	p.vettedMutex.Lock()
	defer p.vettedMutex.Unlock()

	delete(p.vettedListeners, int(s))
}

func (p *Allocator) InvokeVettedListeners(minerID string) {
	p.vettedMutex.RLock()
	defer p.vettedMutex.RUnlock()

	for _, f := range p.vettedListeners {
		go f(minerID)
	}
}
