package hashrate

import (
	"math"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"go.uber.org/atomic"
)

const EMA_INTERVAL = 5 * time.Minute

// const EMA_INTERVAL = 30

type Hashrate struct {
	ema         *Counter
	totalHashes atomic.Uint64
	log         interfaces.ILogger
}

func NewHashrate(log interfaces.ILogger, emaInterval time.Duration) *Hashrate {
	return &Hashrate{
		ema: New(emaInterval),
		log: log,
	}
}

func (h *Hashrate) OnSubmit(diff int64) {
	h.ema.Add(float64(diff))
	h.totalHashes.Add(uint64(diff))
}

func (h *Hashrate) GetTotalHashes() uint64 {
	return h.totalHashes.Load()
}

func (h *Hashrate) GetHashrateGHS() int {
	return int(h.getHashrateHS() / uint64(math.Pow10(9)))
}

func (h *Hashrate) getHashrateHS() uint64 {
	return uint64(h.ema.ValuePer(time.Second)) * uint64(math.Pow(2, 32))
}
