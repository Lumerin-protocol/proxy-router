package hashrate

import (
	"math"
	"sync"
	"time"
)

type emaPrimed struct {
	halfLife  time.Duration
	lastValue float64
	lastTime  time.Time
	mutex     sync.RWMutex

	startedAt     time.Time
	primerSum     float64 // sum of observations during priming period
	primedObsLeft int     // observation left to collect the initial data for ema function primer
}

// NewEmaPrimed creates an EMA counter with the given half-life to be primed
// with arithmetic average of obsCount first observations
func NewEmaPrimed(halfLife time.Duration, obsCount int) *emaPrimed {
	return &emaPrimed{
		halfLife:      halfLife,
		primedObsLeft: obsCount,
	}
}

// Value returns the current value of the counter.
func (c *emaPrimed) Value() float64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.value()
}

// LastValue returns last value of a counter excluding the value decay
func (c *emaPrimed) LastValue() float64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.valueAfter(0)
}

// ValuePer returns the current value of the counter, normalized to the given
// interval. It is actually a Value() * interval / avgInterval.
func (c *emaPrimed) ValuePer(interval time.Duration) float64 {
	return c.Value() * float64(interval) / float64(c.getAvgInterval())
}

func (c *emaPrimed) LastValuePer(interval time.Duration) float64 {
	return c.valueAfter(0) * float64(interval) / float64(c.getAvgInterval())
}

// Add adds a new value to the counter.
func (c *emaPrimed) Add(v float64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.startedAt.IsZero() {
		c.startedAt = getNow()
	}

	if c.primedObsLeft > 0 {
		c.primedObsLeft--
		c.primerSum += v
		if c.primedObsLeft == 0 {
			elapsed := getNow().Sub(c.startedAt)
			c.lastValue = c.primerSum * (float64(c.halfLife) / float64(elapsed))
			c.lastTime = c.startedAt
		}
		return
	}

	c.lastValue = c.value() + v
	c.lastTime = getNow()
}

// Private methods

func (c *emaPrimed) value() float64 {
	return c.valueAfter(getNow().Sub(c.lastTime))
}

// calculates value decay
func (c *emaPrimed) valueAfter(elapsed time.Duration) float64 {
	if c.lastValue == 0 {
		return 0
	}
	w := math.Exp(-float64(elapsed) / float64(c.getAvgInterval()))
	return c.lastValue * w
}

func (c *emaPrimed) getAvgInterval() time.Duration {
	return c.halfLife
}
