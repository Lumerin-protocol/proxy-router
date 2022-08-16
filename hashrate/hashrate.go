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
	h.log.Debugf("new submit: diff %d hashrate %.3f TH/s", diff, float64(h.GetHashrate())/float64(math.Pow10(12)))
}

func (h *Hashrate) GetHashrate() int64 {
	return int64(h.ema.ValuePer(time.Second)) * int64(math.Pow(2, 32))
}

func (h *Hashrate) GetTotalHashes() uint64 {
	return h.totalHashes.Load()
}
