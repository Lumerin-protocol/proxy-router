package allocator

import (
	"context"
	"net/url"
	"sync"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	h "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy"
)

const (
	MINER_VETTING_PERIOD = 3 * time.Minute
)

type Task struct {
	Dest                 *url.URL
	RemainingJobToSubmit float64
	OnSubmit             func(diff float64, ID string)
}

// Scheduler is a proxy wrapper that can schedule one-time tasks to different destinations
type Scheduler struct {
	proxy             StratumProxyInterface
	hashrateCounterID string
	primaryDest       *url.URL
	tasks             lib.Stack[Task]
	totalTaskJob      float64
	newTaskSignal     chan struct{}
	resetTasksSignal  chan struct{}
	log               interfaces.ILogger
}

func NewScheduler(proxy StratumProxyInterface, hashrateCounterID string, defaultDest *url.URL, log interfaces.ILogger) *Scheduler {
	return &Scheduler{
		proxy:             proxy,
		primaryDest:       defaultDest,
		hashrateCounterID: hashrateCounterID,
		log:               log,
		newTaskSignal:     make(chan struct{}, 1),
	}
}

func (p *Scheduler) GetID() string {
	return p.proxy.GetID()
}

func (p *Scheduler) Run(ctx context.Context) error {
	err := p.proxy.Connect(ctx)
	if err != nil {
		return err // handshake error
	}

	proxyTask := lib.NewTaskFunc(p.proxy.Run)
	proxyTask.Start(ctx)

	select {
	case <-ctx.Done():
		<-proxyTask.Done()
		return ctx.Err()
	case <-proxyTask.Done():
		return proxyTask.Err()
	default:
	}

	p.primaryDest = p.proxy.GetDest()

	for {
		// do tasks
		for {
			p.resetTasksSignal = make(chan struct{})
			task, ok := p.tasks.Peek()
			if !ok {
				break
			}
			p.totalTaskJob -= task.RemainingJobToSubmit
			jobDone := make(chan struct{})
			jobDoneOnce := sync.Once{}

			p.log.Debugf("start doing task %s, for job %.0f", task.Dest.String(), task.RemainingJobToSubmit)

			err := p.proxy.SetDest(ctx, task.Dest, func(diff float64) {
				task.RemainingJobToSubmit -= diff
				task.OnSubmit(diff, p.proxy.GetID())
				if task.RemainingJobToSubmit <= 0 {
					jobDoneOnce.Do(func() {
						p.log.Debugf("finished doing task %s, for job %.0f", task.Dest.String(), task.RemainingJobToSubmit)
						close(jobDone)
					})
				}
			})
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				<-proxyTask.Done()
				return ctx.Err()
			case <-proxyTask.Done():
				return proxyTask.Err()
			case <-p.resetTasksSignal:
				close(jobDone)
			case <-jobDone:
			}

			p.tasks.Pop()
		}

		select {
		case <-ctx.Done():
			<-proxyTask.Done()
			return ctx.Err()
		case <-proxyTask.Done():
			return proxyTask.Err()
		case <-p.newTaskSignal:
			continue
		default:
		}

		// remaining time serve default destination
		err := p.proxy.SetDest(ctx, p.primaryDest, nil)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			<-proxyTask.Done()
			return ctx.Err()
		case <-proxyTask.Done():
			return proxyTask.Err()
		case <-p.newTaskSignal:
		}
	}
}

func (p *Scheduler) AddTask(dest *url.URL, jobSubmitted float64, onSubmit func(diff float64, ID string)) {
	shouldSignal := p.tasks.Size() == 0
	p.tasks.Push(Task{
		Dest:                 dest,
		RemainingJobToSubmit: jobSubmitted,
		OnSubmit:             onSubmit,
	})
	p.totalTaskJob += jobSubmitted
	if shouldSignal {
		p.newTaskSignal <- struct{}{}
	}
	p.log.Debugf("added new task, dest: %s, for jobSubmitted: %.0f, totalTaskJob: %.0f", dest, jobSubmitted, p.totalTaskJob)
}

func (p *Scheduler) ResetTasks() {
	p.tasks.Clear()
	close(p.resetTasksSignal)
}

func (p *Scheduler) GetTaskCount() int {
	return p.tasks.Size()
}

func (p *Scheduler) GetTotalTaskJob() float64 {
	return p.totalTaskJob
}

func (p *Scheduler) IsFree() bool {
	return p.tasks.Size() == 0
}

// AcceptsTasks returns true if there are vacant space for tasks for provided interval
func (p *Scheduler) IsAcceptingTasks(duration time.Duration) bool {
	totalJob := 0.0
	for _, tsk := range p.tasks {
		totalJob += tsk.RemainingJobToSubmit
	}
	maxJob := h.GHSToJobSubmitted(p.HashrateGHS()) * duration.Seconds()
	return p.tasks.Size() > 0 && totalJob < maxJob
}

func (p *Scheduler) AddHashrate(dest *url.URL, hrGHS float64, onSubmit func(diff float64, ID string)) {
	p.AddTask(dest, h.GHSToJobSubmitted(hrGHS), onSubmit)
}

func (p *Scheduler) SetPrimaryDest(dest *url.URL) {
	p.primaryDest = dest
	p.newTaskSignal <- struct{}{}
}

func (p *Scheduler) HashrateGHS() float64 {
	hr, ok := p.proxy.GetHashrate().GetHashrateAvgGHSCustom(p.hashrateCounterID)
	if !ok {
		panic("hashrate counter not found")
	}
	return hr
}

func (p *Scheduler) GetStatus() MinerStatus {
	if p.IsVetting() {
		return MinerStatusVetting
	}

	if p.IsFree() {
		return MinerStatusFree
	}

	return MinerStatusBusy
}

func (p *Scheduler) GetCurrentDifficulty() float64 {
	return p.proxy.GetDifficulty()
}

func (p *Scheduler) GetCurrentDest() *url.URL {
	return p.proxy.GetDest()
}

func (p *Scheduler) GetWorkerName() string {
	return p.proxy.GetSourceWorkerName()
}

func (p *Scheduler) GetConnectedAt() time.Time {
	return p.proxy.GetMinerConnectedAt()
}

func (p *Scheduler) GetStats() interface{} {
	return p.proxy.GetStats()
}

func (p *Scheduler) IsVetting() bool {
	return p.GetUptime() < MINER_VETTING_PERIOD
}

func (p *Scheduler) GetUptime() time.Duration {
	return time.Since(p.proxy.GetMinerConnectedAt())
}

func (p *Scheduler) GetDestConns() *map[string]string {
	return p.proxy.GetDestConns()
}

func (p *Scheduler) GetHashrate() proxy.Hashrate {
	return p.proxy.GetHashrate()
}
