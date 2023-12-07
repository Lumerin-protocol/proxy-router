package allocator

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy"
	"go.uber.org/atomic"
)

var (
	ErrConnPrimary           = errors.New("failed to connect to primary dest")
	ErrConnDest              = errors.New("failed to connect to dest")
	ErrProxyExited           = errors.New("proxy exited")
	ErrTaskDeadlineExceeded  = errors.New("task deadline exceeded")
	ErrTaskMinerDisconnected = errors.New("miner disconnected")
)

// Scheduler is a proxy wrapper that can schedule one-time tasks to different destinations
type Scheduler struct {
	// config
	minerVettingShares int
	hashrateCounterID  string

	// state
	primaryDest     *url.URL
	tasks           *TaskList
	newTaskSignal   chan struct{}
	usedHR          *hashrate.Hashrate
	isDisconnecting *atomic.Bool

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
		tasks:              NewTaskList(),
		usedHR:             hashrateFactory(),
		proxy:              proxy,
		onVetted:           onVetted,
		isDisconnecting:    atomic.NewBool(false),
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

	p.primaryDest = p.proxy.GetDest()
	p.log = p.log.Named(fmt.Sprintf("SCH %s %s", p.proxy.GetSourceWorkerName(), lib.ParsePort(p.proxy.GetID())))

	p.log.Infof("proxy connected")

	for {
		if p.proxy.GetDest().String() != p.primaryDest.String() {
			err := p.proxy.SetDestWithoutAutoread(ctx, p.primaryDest, nil)

			if err != nil {
				err := lib.WrapError(ErrConnPrimary, err)
				p.log.Warnf("%s: %s", err, p.primaryDest)
				p.onDisconnect()
				return err
			}
		}
		proxyTask := lib.NewTaskFunc(p.proxy.Run)

		// go func() {
		// 	select {
		// 	case <-ctx.Done():
		// 		return
		// 	case <-proxyTask.Done():
		// 		return
		// 	case <-p.proxy.VettingDone():
		// 	}

		// 	p.log.Infof("vetting done")
		// 	if p.onVetted != nil {
		// 		p.onVetted(p.proxy.GetID())
		// 		p.onVetted = nil
		// 	}
		// }()

		proxyTask.Start(ctx)

		select {
		case <-proxyTask.Done():
			p.onDisconnect()
			return proxyTask.Err()
		default:
		}

		err = p.mainLoop(ctx, proxyTask)
		if errors.Is(err, proxy.ErrDest) || errors.Is(err, proxy.ErrConnectDest) {
			if p.tasks.taskTaken {
				p.tasks.UnlockAndRemove()
			}
			p.log.Warnf("dest error: %v", err)
			p.log.Debugf("reconnecting to primary dest %s", p.primaryDest)
			continue
		} else {
			p.onDisconnect()
			return err
		}
	}
}

func (p *Scheduler) mainLoop(ctx context.Context, proxyTask *lib.Task) error {
	for {
		// do tasks
		proxyExited, err := p.taskLoop(ctx, proxyTask)
		if proxyExited {
			return err
		}

		select {
		case <-proxyTask.Done():
			p.log.Infof("proxy exited: %v", proxyTask.Err())
			return proxyTask.Err()
		case <-p.newTaskSignal:
			continue
		default:
		}

		// all tasks are done, switch to default destination
		err = p.proxy.SetDest(ctx, p.primaryDest, nil)
		if err != nil {
			err = lib.WrapError(ErrConnPrimary, err)
			return err
		}

		select {
		case <-proxyTask.Done():
			p.log.Infof("proxy exited: %v", proxyTask.Err())
			return proxyTask.Err()
		case <-p.newTaskSignal:
		}
	}
}

// taskLoop is a loop that runs tasks until there are no tasks left
func (p *Scheduler) taskLoop(ctx context.Context, proxyTask *lib.Task) (proxyExited bool, err error) {
	for {
		task, ok := p.tasks.LockNextTask()
		if !ok {
			return false, nil
		}

		deadlineCh := time.After(time.Until(task.Deadline))

		p.log.Debugf("start doing task for job ID %s, for job amount %.f", lib.StrShort(task.ID), task.Job)

		select {
		case <-proxyTask.Done():
			err = lib.WrapError(ErrProxyExited, proxyTask.Err())
			task.OnEnd(p.ID(), p.HashrateGHS(), float64(task.RemainingJobToSubmit.Load()), err)
			return true, err
		case <-task.cancelCh:
			p.log.Debugf("task cancelled %s", lib.StrShort(task.ID))
			task.OnEnd(p.ID(), p.HashrateGHS(), float64(task.RemainingJobToSubmit.Load()), nil)
			p.tasks.UnlockAndRemove()
			continue
		case <-deadlineCh:
			err := lib.WrapError(ErrTaskDeadlineExceeded, fmt.Errorf("%s", lib.StrShort(task.ID)))
			p.log.Debugf(err.Error())
			task.OnEnd(p.ID(), p.HashrateGHS(), float64(task.RemainingJobToSubmit.Load()), err)
			p.tasks.UnlockAndRemove()
			continue
		default:
		}

		onSubmit := func(diff float64) {
			task.OnSubmit(diff, p.proxy.GetID())
			p.usedHR.OnSubmit(diff)
			remainingJob := task.RemainingJobToSubmit.Add(-int64(diff))
			if remainingJob <= 0 {
				ok := task.Cancel()
				if ok {
					p.log.Debugf("miner %s finished doing task for job %s", p.ID(), lib.StrShort(task.ID))
				}
			}
		}

		err := p.proxy.SetDest(ctx, task.Dest, onSubmit)
		if err != nil {
			err = lib.WrapError(ErrConnDest, err)
			task.OnEnd(p.ID(), p.HashrateGHS(), float64(task.RemainingJobToSubmit.Load()), err)
			return true, err
		}

		select {
		case <-proxyTask.Done():
			err = lib.WrapError(ErrProxyExited, proxyTask.Err())
			task.OnEnd(p.ID(), p.HashrateGHS(), float64(task.RemainingJobToSubmit.Load()), err)
			return true, err
		case <-task.cancelCh:
			p.log.Debugf("task cancelled %s", lib.StrShort(task.ID))
			task.OnEnd(p.ID(), p.HashrateGHS(), float64(task.RemainingJobToSubmit.Load()), nil)
			p.tasks.UnlockAndRemove()
			continue
		case <-deadlineCh:
			err := lib.WrapError(ErrTaskDeadlineExceeded, fmt.Errorf("%s", lib.StrShort(task.ID)))
			p.log.Debugf(err.Error())
			task.OnEnd(p.ID(), p.HashrateGHS(), float64(task.RemainingJobToSubmit.Load()), err)
			p.tasks.UnlockAndRemove()
			continue
		}
	}
}

func (p *Scheduler) onDisconnect() {
	p.isDisconnecting.Store(true)

	p.tasks.Range(func(task *MinerTask) bool {
		p.log.Debugf("signalling task %s on disconnect", lib.StrShort(task.ID))
		task.OnDisconnect(p.ID(), p.HashrateGHS(), float64(task.RemainingJobToSubmit.Load()))
		task.OnEnd(p.ID(), p.HashrateGHS(), float64(task.RemainingJobToSubmit.Load()), ErrTaskMinerDisconnected)
		return true
	})

	if p.tasks.taskTaken {
		p.tasks.UnlockAndRemove()
	}
}

func (p *Scheduler) getExpectedCycleJob(cycleDuration time.Duration) float64 {
	return hashrate.GHSToJobSubmittedV2(p.HashrateGHS(), cycleDuration)
}

// Scheduler setters protected by mutex

// AddTask adds new task to the queue
func (p *Scheduler) AddTask(
	ID string,
	dest *url.URL,
	jobSubmitted float64,
	onSubmit OnSubmitCb,
	onDisconnect OnDisconnectCb,
	onEnd OnEndCb,
	deadline time.Time,
) {
	newLength := p.tasks.Add(ID, dest, jobSubmitted, deadline, onSubmit, onDisconnect, onEnd)

	if newLength == 1 {
		p.newTaskSignal <- struct{}{}
	}

	taskGHS := hashrate.JobSubmittedToGHSV2(jobSubmitted, time.Until(deadline))
	p.log.Debugf(`added new task, 
	contractID: %s, for jobSubmitted: %.0f, and duration: %s,
	hashrate %0.f, where miners hashrate is %0.f`,
		lib.StrShort(ID), jobSubmitted, time.Until(deadline), taskGHS, p.HashrateGHS())
}

func (p *Scheduler) RemoveTasksByID(ID string) {
	p.tasks.Cancel(ID)
}

// SetPrimaryDest is not protected by mutex
func (p *Scheduler) SetPrimaryDest(dest *url.URL) {
	p.primaryDest = dest
	p.newTaskSignal <- struct{}{}
}

// Scheduler getters protected by mutex

func (p *Scheduler) GetTaskCount() int {
	return p.tasks.Size()
}

func (p *Scheduler) GetTasksByID(ID string) []*MinerTask {
	var tasks []*MinerTask

	p.tasks.Range(func(task *MinerTask) bool {
		if task.ID == ID {
			tasks = append(tasks, task)
		}
		return true
	})

	return tasks
}

func (p *Scheduler) IsFree() bool {
	return p.tasks.Size() == 0
}

func (p *Scheduler) IsPartialBusy(cycleDuration time.Duration) bool {
	return p.tasks.Size() > 0 && p.GetTotalScheduledJob() < p.getExpectedCycleJob(cycleDuration)
}

func (p *Scheduler) IsBusy(cycleDuration time.Duration) bool {
	return p.tasks.Size() > 0 && p.GetTotalScheduledJob() >= p.getExpectedCycleJob(cycleDuration)
}

// AcceptsTasks returns true if there are vacant space for tasks for provided interval
func (p *Scheduler) IsAcceptingTasks(duration time.Duration) bool {
	return !p.IsBusy(duration)
}

func (p *Scheduler) GetTotalScheduledJob() float64 {
	totalJob := 0.0
	p.tasks.Range(func(task *MinerTask) bool {
		totalJob += task.RemainingJob()
		return true
	})
	return totalJob
}

func (p *Scheduler) GetJobCouldBeScheduledTill(interval time.Duration) float64 {
	if interval == 0 {
		return 0
	}
	return p.getExpectedCycleJob(interval) - p.GetTotalScheduledJob()
}

func (p *Scheduler) GetDestinations(cycleDuration time.Duration) []*DestItem {
	return make([]*DestItem, 0)
	// Temporary removing current destinations from the response to avoid locking
	// TODO: reiplement it unsing atomic view to avoid locking

	// dests := make([]*DestItem, 0)
	// cycleJob := p.getExpectedCycleJob(cycleDuration)

	// p.tasks.Range(func(task *MinerTask) bool {
	// 	dests = append(dests, &DestItem{
	// 		Dest:     task.Dest.String(),
	// 		Job:      float64(task.RemainingJobToSubmit.Load()),
	// 		Fraction: task.Job / cycleJob,
	// 	})
	// 	return true
	// })

	// return dests
}

// Data from proxy

// HashrateGHS returns hashrate in GHS
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
	if p.isDisconnecting.Load() {
		return MinerStatusDisconnecting
	}

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

func (p *Scheduler) IsDisconnecting() bool {
	return p.isDisconnecting.Load()
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
