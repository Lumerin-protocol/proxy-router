package allocator

import (
	"testing"
	"time"

	"github.com/Lumerin-protocol/proxy-router/internal/testlib"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func TestTasklistAdd(t *testing.T) {
	repeats := 10000

	tl := NewTaskList()
	testlib.RepeatConcurrent(t, repeats, func(t *testing.T) {
		tl.Add("", nil, 0, time.Now(), nil, nil, nil)
	})

	require.Equal(t, tl.Size(), repeats)
}

func TestTasklistRemove(t *testing.T) {
	repeats := 10000

	tl := NewTaskList()
	startCh := make(chan struct{})

	testlib.RepeatConcurrent(t, repeats, func(t *testing.T) {
		tl.Add("", nil, 0, time.Now(), nil, nil, nil)
		select {
		case <-startCh:
		default:
			close(startCh)
		}
	})

	<-startCh
	testlib.Repeat(t, repeats, func(t *testing.T) {
		_, ok := tl.LockNextTask()
		if ok {
			tl.UnlockAndRemove()
		}
	})

	require.Equal(t, tl.Size(), 0)
}

func TestConditionalChannelCloseConcurrency(t *testing.T) {
	ch := make(chan struct{})
	var isCancelled atomic.Bool

	f := func() bool {
		if isCancelled.CompareAndSwap(false, true) {
			close(ch)
			return true
		}
		return false
	}

	times := 50000
	testlib.Repeat(t, times, func(t *testing.T) {
		ch = make(chan struct{})
		testlib.RepeatConcurrent(t, 4, func(t *testing.T) {
			f()
		})
	})
}
