package allocator

import (
	"context"
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
}

type ListenerHandle int

type MinerItemJobScheduled struct {
	ID       string
	Job      float64
	Fraction float64
}

type MinerIDJob = map[string]float64

type AllocatorInterface interface {
	Run(ctx context.Context) error
	UpsertAllocation(ID string, hashrate float64, dest string, counter func(diff float64)) error
}

type Allocator struct {
	proxies *lib.Collection[*Scheduler]

	lastListenerID  int
	vettedListeners map[int]func(ID string)
	mutex           sync.RWMutex

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
	onSubmit func(diff float64, ID string),
	onDisconnect func(ID string, HrGHS float64, remainingJob float64),
) (minerIDs []string, deltaGHS float64) {
	miners := p.GetFreeMiners()
	p.log.Infof("available free miners %v", miners)

	sort.Slice(miners, func(i, j int) bool {
		return miners[i].HrGHS < miners[j].HrGHS
	})

	for _, miner := range miners {
		minerGHS := miner.HrGHS
		if minerGHS <= hrGHS && minerGHS > 0 {
			proxy, ok := p.proxies.Load(miner.ID)
			if ok {
				proxy.AddTask(ID, dest, hashrate.GHSToJobSubmittedV2(minerGHS, duration), onSubmit, onDisconnect)
				minerIDs = append(minerIDs, miner.ID)
				hrGHS -= minerGHS
				p.log.Infof("full miner %s allocated for %.0f GHS", miner.ID, minerGHS)
			}
		}
	}

	return minerIDs, hrGHS
}

func (p *Allocator) AllocatePartialForHR(
	ID string,
	hrGHS float64,
	dest *url.URL,
	cycleDuration time.Duration,
	onSubmit func(diff float64, ID string),
	onDisconnect func(ID string, hrGHS float64, remainingJob float64),
) (minerIDJob MinerIDJob, remainderGHS float64) {
	jobNeeded := hashrate.GHSToJobSubmitted(hrGHS) * cycleDuration.Seconds()
	minerIDJob, remainderJob := p.AllocatePartialForJob(ID, jobNeeded, dest, cycleDuration, onSubmit, onDisconnect)
	remainderGHS = hashrate.JobSubmittedToGHS(remainderJob) / cycleDuration.Seconds()
	return minerIDJob, remainderGHS
}

func (p *Allocator) AllocatePartialForJob(
	ID string,
	jobNeeded float64,
	dest *url.URL,
	cycleDuration time.Duration,
	onSubmit func(diff float64, ID string),
	onDisconnect func(ID string, hrGHS float64, remainingJob float64),
) (minerIDJob MinerIDJob, remainderGHS float64) {
	p.log.Infof("attemoting to partially allocate job %.f", jobNeeded)

	partialMiners := p.GetPartialMiners(cycleDuration)
	p.log.Infof("available partial miners %v", partialMiners)
	minerIDJob = MinerIDJob{}

	minJob := 50000.0

	for _, miner := range partialMiners {
		minerJobRemaining := miner.Job / miner.Fraction
		// try to add the whole chunk and return
		if minerJobRemaining >= jobNeeded {
			m, ok := p.proxies.Load(miner.ID)
			if ok {
				m.AddTask(ID, dest, jobNeeded, onSubmit, onDisconnect)
				minerIDJob[miner.ID] = jobNeeded
				return minerIDJob, 0
			}
		}
		// try to add at leas a minJob and continue
		if minerJobRemaining >= minJob {
			m, ok := p.proxies.Load(miner.ID)
			if ok {
				m.AddTask(ID, dest, minerJobRemaining, onSubmit, onDisconnect)
				minerIDJob[miner.ID] = jobNeeded
				jobNeeded -= minerJobRemaining
			}
		}
	}

	// search in free miners
	// missing loop cause we already checked full miners
	freeMiners := p.GetFreeMiners()
	p.log.Infof("available free miners %v", freeMiners)

	for _, miner := range freeMiners {
		minerJobRemaining := hashrate.GHSToJobSubmitted(miner.HrGHS) * cycleDuration.Seconds()
		if minerJobRemaining >= jobNeeded {
			m, ok := p.proxies.Load(miner.ID)
			if ok {
				m.AddTask(ID, dest, jobNeeded, onSubmit, onDisconnect)
				minerIDJob[miner.ID] = jobNeeded
				return minerIDJob, 0
			}
		}
	}

	return minerIDJob, jobNeeded
}

func (p *Allocator) GetFreeMiners() []MinerItem {
	freeMiners := []MinerItem{}
	p.proxies.Range(func(item *Scheduler) bool {
		if item.IsVetting() {
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

func (p *Allocator) GetPartialMiners(contractCycleDuration time.Duration) []MinerItemJobScheduled {
	partialMiners := []MinerItemJobScheduled{}
	p.proxies.Range(func(item *Scheduler) bool {
		if item.IsVetting() {
			return true
		}
		if item.IsAcceptingTasks(contractCycleDuration) {
			job := item.GetTotalTaskJob() * HashratePredictionAdjustment
			fraction := hashrate.JobSubmittedToGHSV2(job, contractCycleDuration) / (item.HashrateGHS() * HashratePredictionAdjustment)

			partialMiners = append(partialMiners, MinerItemJobScheduled{
				ID:       item.ID(),
				Job:      job,
				Fraction: fraction,
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

func (p *Allocator) GetMinersFulfillingContract(contractID string) []*DestItem {
	dests := []*DestItem{}
	p.GetMiners().Range(func(item *Scheduler) bool {
		if item.IsVetting() {
			return true
		}
		tasks := item.GetTasksByID(contractID)
		for _, task := range tasks {
			dests = append(dests, &DestItem{
				Dest: task.Dest.String(),
				Job:  float64(task.RemainingJobToSubmit.Load()),
			})
		}
		return true
	})
	return dests
}

// CancelTasks cancels all tasks for a specified contractID
func (p *Allocator) CancelTasks(contractID string) {
	p.GetMiners().Range(func(item *Scheduler) bool {
		item.RemoveTasksByID(contractID)
		return true
	})
}

func (p *Allocator) AddVettedListener(f func(ID string)) ListenerHandle {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ID := p.lastListenerID
	p.lastListenerID++

	p.vettedListeners[ID] = f
	return ListenerHandle(ID)
}

func (p *Allocator) RemoveVettedListener(s ListenerHandle) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.vettedListeners, int(s))
}

func (p *Allocator) InvokeVettedListeners(minerID string) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for _, f := range p.vettedListeners {
		go f(minerID)
	}
}
