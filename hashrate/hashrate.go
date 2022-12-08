package hashrate

import (
	"math"
	"time"

	"go.uber.org/atomic"
)

type Hashrate struct {
	emaBase     Counter
	ema5m       Counter
	ema30m      Counter
	ema1h       Counter
	totalHashes atomic.Uint64
}

func NewHashrate() *Hashrate {
	return &Hashrate{
		emaBase: NewEmaPrimed(3*time.Minute, 10),
		ema5m:   NewEma(5 * time.Minute),
		ema30m:  NewEma(30 * time.Minute),
		ema1h:   NewEma(1 * time.Hour),
	}
}

func (h *Hashrate) OnSubmit(diff int64) {
	h.emaBase.Add(float64(diff))
	h.ema5m.Add(float64(diff))
	h.ema30m.Add(float64(diff))
	h.ema1h.Add(float64(diff))
	h.totalHashes.Add(uint64(diff))
}

func (h *Hashrate) GetTotalHashes() uint64 {
	return h.totalHashes.Load()
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
	hashrateHS := uint64(averagePerSecond) * uint64(math.Pow(2, 32))
	return HSToGHS(hashrateHS)
}

func HSToGHS(hashrateHS uint64) int {
	return int(hashrateHS / uint64(math.Pow10(9)))
}
