package hashrate

import (
	"math"
	"sync"
	"time"
)

type emaPrimed struct {
	avgInterval time.Duration
	lastValue   float64
	lastTime    time.Time
	startedAt   time.Time
	lk          sync.RWMutex

	initSum       float64
	primedObsLeft int
}

// NewEmaPrimed creates an EMA counter with the given avgInterval to be primed
// with arithmetic average of obsCount first observations
func NewEmaPrimed(avgInterval time.Duration, obsCount int) *emaPrimed {
	return &emaPrimed{
		avgInterval:   avgInterval,
		primedObsLeft: obsCount,
	}
}

// Value returns the current value of the counter.
func (c *emaPrimed) Value() float64 {
	c.lk.RLock()
	defer c.lk.RUnlock()
	return c.value()
}

// LastValue returns last value of a counter excluding the value decay
func (c *emaPrimed) LastValue() float64 {
	c.lk.RLock()
	defer c.lk.RUnlock()
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
	c.lk.Lock()
	defer c.lk.Unlock()

	if c.startedAt.IsZero() {
		c.startedAt = getNow()
	}

	if c.primedObsLeft > 0 {
		c.primedObsLeft--
		c.initSum += v
		if c.primedObsLeft == 0 {
			elapsed := getNow().Sub(c.startedAt)
			c.lastValue = c.initSum * (float64(c.avgInterval) / float64(elapsed))
			c.lastTime = c.startedAt
		} else {
			return

		}
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
	return c.avgInterval
}
