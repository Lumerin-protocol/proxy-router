package hashrate

import (
	"context"
	"math"
	"math/big"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

const EMA_INTERVAL = 600

// const EMA_INTERVAL = 30

//an individual validator which will operate as a thread
type Hashrate struct {
	StartTime      time.Time
	Hashrates      []int
	HashesAnalyzed uint
	diff           int64
	lastCalc       time.Time

	log interfaces.ILogger
}

func NewHashrate(log interfaces.ILogger) *Hashrate {
	return &Hashrate{
		StartTime:      time.Now(),
		HashesAnalyzed: 0,
		log:            log,
	}
}

func (h *Hashrate) OnSubmit(workerName, nonce, nTime string) {
	h.HashesAnalyzed++
	h.log.Debugf("==========>hashes analyzed: %d", h.HashesAnalyzed)
}

func (h *Hashrate) OnSetDefficulty(diff int) {
	if h.diff == 0 {
		h.diff = int64(diff)
		h.lastCalc = time.Now()
		return
	}

	timePassed := time.Since(h.lastCalc).Seconds()
	timeRatio := timePassed / EMA_INTERVAL

	alpha := 1 - 1.0/math.Exp(timeRatio)
	r := int64(alpha*float64(diff) + (1-alpha)*float64(h.diff))

	h.diff = r
	h.lastCalc = time.Now()
}

func (h *Hashrate) Run(ctx context.Context) error {
	for {
		startHashCount := h.HashesAnalyzed
		timeInterval := time.Second * EMA_INTERVAL

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(timeInterval):
		}

		// calculate 5 minute moving average of hashrate
		endHashCount := h.HashesAnalyzed
		hashesAnalyzed := endHashCount - startHashCount

		if h.diff == 0 {
			h.log.Debugf("pool difficulty not yet defined: skipping hashrate calculation")
			continue
		}

		h.log.Debugf("Current Pool Difficulty: %d", h.diff)
		h.log.Debugf("Current Hashes Analyzed in this interval: %d", hashesAnalyzed)

		//calculate the number of hashes represented by the pool difficulty target
		bigDiffTarget := big.NewInt(int64(h.diff))
		bigHashesAnalyzed := big.NewInt(int64(hashesAnalyzed))

		result := new(big.Int).Exp(big.NewInt(2), big.NewInt(32), nil)
		hashesPerSubmission := new(big.Int).Mul(bigDiffTarget, result)
		totalHashes := new(big.Int).Mul(hashesPerSubmission, bigHashesAnalyzed)

		//divide represented hashes by time duration
		rateBigInt := new(big.Int).Div(totalHashes, big.NewInt(int64(timeInterval.Seconds())))
		hashrate := int(rateBigInt.Int64())

		// take average hourly average of hashrate
		if len(h.Hashrates) >= 6 {
			h.Hashrates = h.Hashrates[1:]
		}
		hashSum := 0
		for _, h := range h.Hashrates {
			hashSum += h
		}
		hashSum += hashrate
		newHashrate := hashSum / (len(h.Hashrates) + 1)
		h.Hashrates = append(h.Hashrates, newHashrate)

		h.log.Debugf("current Hashrate Moving Average %d", newHashrate)
	}
}

func (h *Hashrate) GetHashrate() int {
	return h.Hashrates[len(h.Hashrates)-1]
}
