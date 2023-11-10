package allocator

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync/atomic"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	h "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy"
)

// Scheduler is a proxy wrapper that can schedule one-time tasks to different destinations
type Scheduler struct {
	// config
	minerVettingShares int
	hashrateCounterID  string
	primaryDest        *url.URL

	// state
	totalTaskJob     float64
	newTaskSignal    chan struct{}
	resetTasksSignal chan struct{}
	tasks            lib.Stack[Task]
	usedHR           *hashrate.Hashrate

	// deps
	proxy    StratumProxyInterface
	onVetted func(ID string)
	log      interfaces.ILogger
}

func NewScheduler(proxy StratumProxyInterface, hashrateCounterID string, defaultDest *url.URL, minerVettingShares int, hashrateFactory HashrateFactory, onVetted func(ID string), log interfaces.ILogger) *Scheduler {
	return &Scheduler{
		primaryDest:        defaultDest,
		hashrateCounterID:  hashrateCounterID,
		minerVettingShares: minerVettingShares,
		newTaskSignal:      make(chan struct{}, 1),
		usedHR:             hashrateFactory(),
		proxy:              proxy,
		onVetted:           onVetted,
		log:                log,
	}
}

func (p *Scheduler) ID() string {
	return p.proxy.GetID()
}

func (p *Scheduler) Run(ctx context.Context) error {
	err := p.proxy.Connect(ctx)
	if err != nil {
		return err // handshake error
	}

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-p.proxy.VettingDone():
		}

		p.log.Infof("miner %s vetting done", p.proxy.GetID())
		if p.onVetted != nil {
			p.onVetted(p.proxy.GetID())
			p.onVetted = nil
		}
	}()

	for {
		if p.proxy.GetDest().String() != p.primaryDest.String() {
			err := p.proxy.ConnectDest(ctx, p.primaryDest)
			if err != nil {
				err := lib.WrapError(fmt.Errorf("failed to connect to primary dest"), err)
				p.log.Warnf("%s: %s", err, p.primaryDest)
				return err
			}
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

		err = p.taskLoop(ctx, proxyTask)
		if errors.Is(err, proxy.ErrDest) || errors.Is(err, proxy.ErrConnectDest) {
			p.log.Warnf("dest error: %s", err)
			p.log.Infof("reconnecting to primary dest %s", p.primaryDest)
			continue
		} else {
			return err
		}
	}
}

func (p *Scheduler) taskLoop(ctx context.Context, proxyTask *lib.Task) error {
	for {
		// do tasks
		for {
			p.resetTasksSignal = make(chan struct{})
			task, ok := p.tasks.Peek()
			if !ok {
				break
			}
			select {
			case <-task.cancelCh:
				p.log.Debugf("task cancelled %s", task.ID)
				p.tasks.Pop()
				continue
			default:
			}

			jobDoneCh := make(chan struct{})
			remainingJob := float64(task.RemainingJobToSubmit.Load())
			p.log.Debugf("start doing task for job ID %s, for job amount %.f", lib.StrShort(task.ID), remainingJob)
			p.totalTaskJob -= remainingJob

			err := p.proxy.SetDest(ctx, task.Dest, func(diff float64) {
				task.OnSubmit(diff, p.proxy.GetID())
				p.usedHR.OnSubmit(diff)
				remainingJob := task.RemainingJobToSubmit.Add(-int64(diff))
				if remainingJob <= 0 {
					select {
					case <-jobDoneCh:
					default:
						p.log.Debugf("finished doing task for job %s", task.ID)
						close(jobDoneCh)
					}
				}
			})
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				<-proxyTask.Done()
				p.signalTasksOnDisconnect()
				return ctx.Err()
			case <-proxyTask.Done():
				p.signalTasksOnDisconnect()
				return proxyTask.Err()
			case <-p.resetTasksSignal:
				close(jobDoneCh)
				p.log.Debugf("tasks resetted")
			case <-task.cancelCh:
				close(jobDoneCh)
				p.log.Debugf("task cancelled %s", task.ID)
			case <-jobDoneCh:
			}

			p.tasks.Pop()
		}

		select {
		case <-ctx.Done():
			<-proxyTask.Done()
			p.signalTasksOnDisconnect()
			return ctx.Err()
		case <-proxyTask.Done():
			p.signalTasksOnDisconnect()
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
			p.signalTasksOnDisconnect()
			<-proxyTask.Done()
			return ctx.Err()
		case <-proxyTask.Done():
			p.signalTasksOnDisconnect()
			return proxyTask.Err()
		case <-p.newTaskSignal:
		}
	}
}

func (p *Scheduler) signalTasksOnDisconnect() {
	for {
		task, ok := p.tasks.Pop()
		if !ok {
			break
		}
		task.OnDisconnect(p.ID(), p.HashrateGHS())
	}
}

func (p *Scheduler) AddTask(
	ID string,
	dest *url.URL,
	jobSubmitted float64,
	onSubmit func(diff float64, ID string),
	onDisconnect func(ID string, HrGHS float64),
) {
	shouldSignal := p.tasks.Size() == 0
	remainingJob := new(atomic.Int64)
	remainingJob.Store(int64(jobSubmitted))

	p.tasks.Push(Task{
		ID:                   ID,
		Dest:                 dest,
		RemainingJobToSubmit: remainingJob,
		OnSubmit:             onSubmit,
		OnDisconnect:         onDisconnect,
		cancelCh:             make(chan struct{}),
	})
	p.totalTaskJob += jobSubmitted
	if shouldSignal {
		p.newTaskSignal <- struct{}{}
	}
	p.log.Debugf("added new task, dest: %s, for jobSubmitted: %.0f, totalTaskJob: %.0f", dest, jobSubmitted, p.totalTaskJob)
}

// TODO: ensure it is concurrency safe
func (p *Scheduler) RemoveTasksByID(ID string) {
	p.tasks.Range(func(task Task) bool {
		if task.ID == ID {
			select {
			case <-task.cancelCh:
			default:
				close(task.cancelCh)
			}
		}
		return true
	})
}

func (p *Scheduler) GetTaskCount() int {
	return p.tasks.Size()
}

func (p *Scheduler) GetTasksByID(ID string) []Task {
	var tasks []Task
	for _, tsk := range p.tasks {
		if tsk.ID == ID {
			tasks = append(tasks, tsk)
		}
	}
	return tasks
}

func (p *Scheduler) GetTotalTaskJob() float64 {
	return p.totalTaskJob
}

func (p *Scheduler) IsFree() bool {
	return p.tasks.Size() == 0
}

func (p *Scheduler) IsPartialBusy(cycleDuration time.Duration) bool {
	return p.totalTaskJob < hashrate.GHSToJobSubmittedV2(p.HashrateGHS(), cycleDuration)
}

// AcceptsTasks returns true if there are vacant space for tasks for provided interval
func (p *Scheduler) IsAcceptingTasks(duration time.Duration) bool {
	totalJob := 0.0
	for _, tsk := range p.tasks {
		totalJob += float64(tsk.RemainingJobToSubmit.Load())
	}
	maxJob := h.GHSToJobSubmittedV2(p.HashrateGHS(), duration)
	return p.tasks.Size() > 0 && totalJob < maxJob
}

func (p *Scheduler) SetPrimaryDest(dest *url.URL) {
	p.primaryDest = dest
	p.newTaskSignal <- struct{}{}
}

func (p *Scheduler) HashrateGHS() float64 {
	if time.Since(p.proxy.GetMinerConnectedAt()) < 10*time.Minute {
		hr, ok := p.proxy.GetHashrate().GetHashrateAvgGHSCustom("mean")
		if !ok {
			panic("hashrate counter not found")
		}
		return hr
	}
	hr, ok := p.proxy.GetHashrate().GetHashrateAvgGHSCustom(p.hashrateCounterID)
	if !ok {
		panic("hashrate counter not found")
	}
	return hr
}

func (p *Scheduler) GetStatus(cycleDuration time.Duration) MinerStatus {
	if p.IsVetting() {
		return MinerStatusVetting
	}

	if p.IsFree() {
		return MinerStatusFree
	}

	if p.IsPartialBusy(cycleDuration) {
		return MinerStatusPartialBusy
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
	return p.proxy.IsVetting()
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

func (p *Scheduler) GetUsedHashrate() proxy.Hashrate {
	return p.usedHR
}

func (p *Scheduler) GetDestinations() []*DestItem {
	dests := make([]*DestItem, p.tasks.Size())

	for i, t := range p.tasks {
		dests[i] = &DestItem{
			Dest: t.Dest.String(),
			Job:  float64(t.RemainingJobToSubmit.Load()),
		}
	}

	return dests
}
