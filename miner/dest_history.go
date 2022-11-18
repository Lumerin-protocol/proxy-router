package miner

import (
	"time"

	"github.com/gammazero/deque"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type DestHistory struct {
	data *deque.Deque[HistoryItem]
	cap  int
}

type HistoryItem struct {
	ContractID string
	Dest       interfaces.IDestination
	Timestamp  time.Time
	Duration   time.Duration
}

// NewDestHistory creates the history data structure. Cap will be rounded up to the nearest power of 2
// When the history log reaches its capacity, the oldest item will be overwritten. The implementation uses Ring buffer
// (deque) to avoid unnecessary allocations
func NewDestHistory(cap int) *DestHistory {
	return &DestHistory{
		data: deque.New[HistoryItem](cap, cap),
		cap:  cap,
	}
}

func (h *DestHistory) Add(dest interfaces.IDestination, contractID string, timestamp *time.Time) {
	if timestamp == nil {
		t := time.Now()
		timestamp = &t
	}

	// sets duration of the previous destination
	if h.data.Len() > 0 {
		recentlyAdded := h.data.Back()
		recentlyAdded.Duration = timestamp.Sub(recentlyAdded.Timestamp)
		h.data.Set(h.data.Len()-1, recentlyAdded)
	}

	if h.data.Len() >= h.cap {
		h.data.PopFront()
	}

	h.data.PushBack(HistoryItem{Dest: dest, Timestamp: *timestamp, ContractID: contractID})
}

func (h *DestHistory) Len() int {
	return h.data.Len()
}

func (h *DestHistory) Get(index int) HistoryItem {
	return h.data.At(index)
}

func (h *DestHistory) Range(f func(item HistoryItem) bool) {
	for i := 0; i < h.data.Len(); i++ {
		shouldContinue := f(h.data.At(i))
		if !shouldContinue {
			return
		}
	}
}

func (h *DestHistory) RangeContractID(contractID string, f func(item HistoryItem) bool) {
	h.Range(func(item HistoryItem) bool {
		if item.ContractID == contractID {
			return f(item)
		}
		return true
	})
}
