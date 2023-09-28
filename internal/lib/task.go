package lib

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

// Task is a wrapper around a function that can be started and stopped
type Task struct {
	runFunc func(ctx context.Context) error

	isRunning atomic.Bool           // bool
	stopCh    atomic.Value          // chan struct{}
	doneCh    atomic.Value          // chan struct{}
	cancel    atomic.Value          // context.CancelFunc
	err       atomic.Pointer[error] // error
	name string
}

type Runnable interface {
	Run(ctx context.Context) error
}

// NewTask creates a new task from Runnable that runs in a separate goroutine that can be started and stopped
func NewTask(runnable Runnable) *Task {
	t := &Task{
		runFunc: runnable.Run,
	}
	t.doneCh.Store(make(chan struct{}))
	return t
}

// NewTaskFunc creates a new task from a function that runs in a separate goroutine that can be started and stopped
func NewTaskFunc(f func(ctx context.Context) error, name string) *Task {
	t := &Task{
		runFunc: f,
		name: name,
	}
	t.doneCh.Store(make(chan struct{}))
	return t
}

func (s *Task) Start(ctx context.Context) {
	if !s.isRunning.CompareAndSwap(false, true) {
		panic("already running")
	}
	subCtx, cancel := context.WithCancel(ctx)
	s.cancel.Store(cancel)

	s.stopCh.Store(make(chan struct{}))

	go func() {
		err := s.runFunc(subCtx)
		isContextErr := errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)

		// returned due to calling Stop()
		if ctx.Err() == nil && subCtx.Err() != nil && isContextErr {
			fmt.Printf("Task %s returned due to calling Stop().", s.name)
			close(s.stopCh.Load().(chan struct{}))
			return
		}

		// returned due to cancelling context from outside
		if ctx.Err() != nil || !isContextErr {
			fmt.Printf("Task %s returned due to cancelling context from outside.", s.name)
			s.err.Store(&err)
			close(s.doneCh.Load().(chan struct{}))
			close(s.stopCh.Load().(chan struct{}))

			return
		}

		// returned due to interal error
		s.err.Store(&err)
		close(s.doneCh.Load().(chan struct{}))
		close(s.stopCh.Load().(chan struct{}))
	}()
}

func (s *Task) Stop() <-chan struct{} {
	if !s.isRunning.CompareAndSwap(true, false) {
		closedChan := make(chan struct{})
		close(closedChan)
		return closedChan
	}
	c := s.cancel.Load()
	if c != nil {
		c.(context.CancelFunc)()
	}
	return s.stopCh.Load().(chan struct{})
}

// Done returns a channel that's closed when task exited or cancelled from outside using context
// When Stop called done is not closed
func (s *Task) Done() <-chan struct{} {
	return s.doneCh.Load().(chan struct{})
}

// Err returns error that caused routine to exit
func (s *Task) Err() error {
	e := s.err.Load()
	if e == nil {
		return nil
	}
	return *e
}
