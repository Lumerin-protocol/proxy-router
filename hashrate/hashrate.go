package hashrate

import (
	"math"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"go.uber.org/atomic"
)

type Hashrate struct {
	ema5m       *Counter
	ema30m      *Counter
	ema1h       *Counter
	custom      map[time.Duration]*Counter
	totalHashes atomic.Uint64
	log         interfaces.ILogger
}

func NewHashrate(log interfaces.ILogger) *Hashrate {
	return &Hashrate{
		ema5m:  New(5 * time.Minute),
		ema30m: New(30 * time.Minute),
		ema1h:  New(1 * time.Hour),
		log:    log,
	}
}

func NewHashrateCustom(log interfaces.ILogger, durations ...time.Duration) *Hashrate {

	instance := &Hashrate{
		ema5m:  New(5 * time.Minute),
		ema30m: New(30 * time.Minute),
		ema1h:  New(1 * time.Hour),
		log:    log,
	}

	if len(durations) > 0 {
		custom := make(map[time.Duration]*Counter, len(durations))
		for _, duration := range durations {
			custom[duration] = New(duration)
		}
		instance.custom = custom
	}

	return instance
}

func (h *Hashrate) OnSubmit(diff int64) {
	diffFloat := float64(diff)

	h.ema5m.Add(diffFloat)
	h.ema30m.Add(diffFloat)
	h.ema1h.Add(diffFloat)

	for _, item := range h.custom {
		item.Add(diffFloat)
	}

	h.totalHashes.Add(uint64(diff))
}

func (h *Hashrate) GetTotalHashes() uint64 {
	return h.totalHashes.Load()
}

// Deprecated: use GetHashrate5minAvgGHS
func (h *Hashrate) GetHashrateGHS() int {
	return h.averageSubmitDiffToGHS(h.ema5m.ValuePer(time.Second))
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

func (h *Hashrate) GetHashrateAvgGHSCustom(avg time.Duration) (hrGHS int, ok bool) {
	ema, ok := h.custom[avg]
	if !ok {
		return 0, false
	}
	return h.averageSubmitDiffToGHS(ema.ValuePer(time.Second)), true
}

// averageSubmitDiffToGHS converts average value provided by ema to hashrate in GH/S
func (h *Hashrate) averageSubmitDiffToGHS(averagePerSecond float64) int {
	return HSToGHS(JobSubmittedToHS(averagePerSecond))
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
