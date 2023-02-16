package hashrate

import (
	"math"
	"time"

	"go.uber.org/atomic"
)

type Hashrate struct {
	emaBase        Counter
	ema5m          Counter
	ema30m         Counter
	ema1h          Counter
	custom         map[time.Duration]Counter
	totalWork      atomic.Uint64
	lastSubmitTime atomic.Int64 // stores last submit time in unix seconds
}

func NewHashrate(durations ...time.Duration) *Hashrate {
	instance := &Hashrate{
		emaBase: NewEma(5 * time.Minute),
		ema5m:   NewEma(5 * time.Minute),
		ema30m:  NewEma(30 * time.Minute),
		ema1h:   NewEma(1 * time.Hour),
	}

	if len(durations) > 0 {
		custom := make(map[time.Duration]Counter, len(durations))
		for _, duration := range durations {
			custom[duration] = NewEma(duration)
		}
		instance.custom = custom
	}

	return instance
}

func NewHashrateV2(counters ...Counter) *Hashrate {
	instance := NewHashrate()

	if len(counters) > 0 {
		custom := make(map[time.Duration]Counter, len(counters))
		for i, counter := range counters {
			custom[time.Duration(i)] = counter
		}
		instance.custom = custom
	}

	return instance
}

func (h *Hashrate) OnSubmit(diff int64) {
	diffFloat := float64(diff)

	h.emaBase.Add(diffFloat)
	h.ema5m.Add(diffFloat)
	h.ema30m.Add(diffFloat)
	h.ema1h.Add(diffFloat)

	h.totalWork.Add(uint64(diff))
	h.setLastSubmitTime(time.Now())

	for _, item := range h.custom {
		item.Add(diffFloat)
	}
}

func (h *Hashrate) GetTotalWork() uint64 {
	return h.totalWork.Load()
}

func (h *Hashrate) GetLastSubmitTime() time.Time {
	return time.Unix(h.lastSubmitTime.Load(), 0)
}

func (h *Hashrate) setLastSubmitTime(t time.Time) {
	h.lastSubmitTime.Store(t.Unix())
}

func (h *Hashrate) GetHashrateGHS() int {
	return h.averageSubmitDiffToGHS(h.emaBase.ValuePer(time.Second))
}

func (h *Hashrate) GetHashrate5minAvgGHS() int {
	return h.averageSubmitDiffToGHS(h.ema5m.ValuePer(time.Second))
}

func (h *Hashrate) GetHashrate30minAvgGHS() int {
	return h.averageSubmitDiffToGHS(h.ema30m.ValuePer(time.Second))
}

func (h *Hashrate) GetHashrate1hAvgGHS() int {
	return h.averageSubmitDiffToGHS(h.ema1h.ValuePer(time.Second))
}

// averageSubmitDiffToGHS converts average value provided by ema to hashrate in GH/S
func (h *Hashrate) averageSubmitDiffToGHS(averagePerSecond float64) int {
	return HSToGHS(JobSubmittedToHS(averagePerSecond))
}

func (h *Hashrate) GetHashrateAvgGHSCustom(avg time.Duration) (hrGHS int, ok bool) {
	ema, ok := h.custom[avg]
	if !ok {
		return 0, false
	}
	return h.averageSubmitDiffToGHS(ema.ValuePer(time.Second)), true
}

func JobSubmittedToHS(jobSubmitted float64) float64 {
	return jobSubmitted * math.Pow(2, 32)
}

func HSToJobSubmitted(hrHS float64) float64 {
	return hrHS / math.Pow(2, 32)
}

func HSToGHS(hashrateHS float64) int {
	return int(hashrateHS / math.Pow10(9))
}

func GHSToHS(hrGHS int) float64 {
	return float64(hrGHS) * math.Pow10(9)
}
