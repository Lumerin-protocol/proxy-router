package allocator

import (
	"context"
	"net/url"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	h "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
)

type Task struct {
	Dest         *url.URL
	JobSubmitted float64
	OnSubmit     func(diff float64)
}

// Scheduler is a proxy wrapper that can schedule one-time tasks to different destinations
type Scheduler struct {
	proxy         StratumProxyInterface
	primaryDest   *url.URL
	tasks         lib.Stack[Task]
	totalTaskJob  float64
	newTaskSignal chan struct{}
	log           interfaces.ILogger
}

func NewScheduler(proxy StratumProxyInterface, defaultDest *url.URL, log interfaces.ILogger) *Scheduler {
	return &Scheduler{
		proxy:         proxy,
		primaryDest:   defaultDest,
		log:           log,
		newTaskSignal: make(chan struct{}, 1),
	}
}

func (p *Scheduler) GetID() string {
	return p.proxy.GetID()
}

func (p *Scheduler) Run(ctx context.Context) error {
	proxyTask := lib.NewTaskFunc(p.proxy.Run)
	proxyTask.Start(ctx)

	select {
	case <-proxyTask.Done():
		return proxyTask.Err()
	case <-p.proxy.HandshakeDoneSignal():
	}

	p.primaryDest = p.proxy.GetDest()

	for {
		// do tasks
		for task, ok := p.tasks.Peek(); ok; {
			p.totalTaskJob -= task.JobSubmitted
			jobDone := make(chan struct{})

			err := p.proxy.SetDest(ctx, task.Dest, func(diff float64) {
				task.JobSubmitted -= diff
				task.OnSubmit(diff)
				if task.JobSubmitted <= 0 {
					close(jobDone)
				}
			})
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-proxyTask.Done():
				return proxyTask.Err()
			case <-jobDone:
			}

			p.tasks.Pop()
		}

		select {
		case <-ctx.Done():
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
			return ctx.Err()
		case <-proxyTask.Done():
			return proxyTask.Err()
		case <-p.newTaskSignal:
		}
	}
}

func (p *Scheduler) AddTask(dest *url.URL, jobSubmitted float64, onSubmit func(diff float64)) {
	p.tasks.Push(Task{
		Dest:         dest,
		JobSubmitted: jobSubmitted,
		OnSubmit:     onSubmit,
	})
	p.totalTaskJob += jobSubmitted
}

func (p *Scheduler) RemoveTasks() {
	p.tasks.Clear()
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
		totalJob += tsk.JobSubmitted
	}
	maxJob := h.GHSToJobSubmitted(p.proxy.GetHashrate()) * duration.Seconds()
	return p.tasks.Size() > 0 && totalJob < maxJob
}

func (p *Scheduler) AddHashrate(dest *url.URL, hrGHS float64, onSubmit func(diff float64)) {
	p.AddTask(dest, h.GHSToJobSubmitted(hrGHS), onSubmit)
}

func (p *Scheduler) SetPrimaryDest(dest *url.URL) {
	p.primaryDest = dest
	p.newTaskSignal <- struct{}{}
}

func (p *Scheduler) HashrateGHS() float64 {
	return p.proxy.GetHashrate()
}

func (p *Scheduler) GetStatus() MinerStatus {
	// if s.IsVetting() {
	// 	return MinerStatusVetting
	// }

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

// if p.totalTaskJob+jobSubmitted > p.getJobPerCycle() {
// 	return false
// }

// func (p *Scheduler) getJobPerCycle() float64 {
// 	return h.GHSToJobSubmitted(int(p.proxy.GetHashrate()))
// }
