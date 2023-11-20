package allocator

import (
	"net/url"
	"sync"
	"time"

	"github.com/gammazero/deque"
	"go.uber.org/atomic"
)

type MinerTask struct {
	ID                   string
	Dest                 *url.URL
	Job                  float64
	RemainingJobToSubmit *atomic.Int64
	cancelCh             chan struct{}
	OnSubmit             func(diff float64, ID string)
	OnDisconnect         func(ID string, HrGHS float64, remainingJob float64)
	Deadline             time.Time
}

func (t *MinerTask) RemainingJob() float64 {
	return float64(t.RemainingJobToSubmit.Load())
}

func (t *MinerTask) Cancel() (firstCancel bool) {
	select {
	case <-t.cancelCh:
		return false
	default:
		close(t.cancelCh)
		return true
	}
}

type TaskList struct {
	tasks     *deque.Deque[*MinerTask]
	mutex     sync.RWMutex
	taskTaken bool
}

func NewTaskList() *TaskList {
	return &TaskList{
		tasks:     deque.New[*MinerTask](),
		mutex:     sync.RWMutex{},
		taskTaken: false,
	}
}

func (p *TaskList) Add(task *MinerTask) (len int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.tasks.PushBack(task)
	return p.tasks.Len()
}

// returns the first element of the task queue
func (p *TaskList) LockNextTask() (t *MinerTask, ok bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.taskTaken {
		panic("task already taken")
	}

	if p.tasks.Len() == 0 {
		return nil, false
	}

	p.taskTaken = true
	return p.tasks.Front(), true
}

// removes lock and removes from the task queue
func (p *TaskList) UnlockAndRemove() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.taskTaken {
		panic("task not taken")
	}
	p.taskTaken = false

	if p.tasks.Len() == 0 {
		panic("no tasks in queue, when there should be at least one")
	}
	p.tasks.PopFront()
}

func (p *TaskList) Unlock() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.taskTaken {
		panic("task not taken")
	}
	p.taskTaken = false
}

func (p *TaskList) Size() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.tasks.Len()
}

func (p *TaskList) CancelAll() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.taskTaken {
		p.tasks.Front().Cancel()
	}

	p.tasks.Clear()
}

func (p *TaskList) Cancel(contractID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for i := 0; i < p.tasks.Len(); i++ {
		if i == 0 && p.taskTaken {
			p.tasks.Front().Cancel()
			continue
		}
		task := p.tasks.At(i)
		if task.ID == contractID {
			p.tasks.Remove(i)
		}
	}
}

func (p *TaskList) Range(f func(task *MinerTask) bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for i := 0; i < p.tasks.Len(); i++ {
		if !f(p.tasks.At(i)) {
			return
		}
	}
}
