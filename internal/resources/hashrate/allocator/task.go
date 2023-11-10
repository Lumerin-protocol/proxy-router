package allocator

import (
	"net/url"
	"sync/atomic"
)

type Task struct {
	ID                   string
	Dest                 *url.URL
	RemainingJobToSubmit *atomic.Int64
	OnSubmit             func(diff float64, ID string)
	OnDisconnect         func(ID string, HrGHS float64, remainingJob float64)
	cancelCh             chan struct{}
}

type DestItem struct {
	Dest string
	Job  float64
}
