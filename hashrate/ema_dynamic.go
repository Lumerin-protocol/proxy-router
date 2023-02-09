package hashrate

import (
	"math"
	"sync"
	"time"
)

type emaDynamic struct {
	avgInterval    time.Duration
	minAvgInterval time.Duration

	lastValue    float64
	lastTime     time.Time
	lastInterval time.Duration
	startedAt    time.Time
	mutex        sync.RWMutex
}

// NewEmaDynamic creates a new EMA counter with dynamic half-life. It decreases the time
// resulting values are averaged out, helping to reach avg value more quick at the beginning
// of the measurment. Dynamic half-life starts from minHalfLife and reaches halfLife
func NewEmaDynamic(halfLife time.Duration, minHalfLife time.Duration) *emaDynamic {
	return &emaDynamic{
		avgInterval:    halfLife,
		minAvgInterval: minHalfLife,
	}
}

// Value returns the current value of the counter.
func (c *emaDynamic) Value() float64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.value()
}

// LastValue returns last value of a counter excluding the value decay
func (c *emaDynamic) LastValue() float64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.valueAfter(0)
}

// ValuePer returns the current value of the counter, normalized to the given
// interval. It is actually a Value() * interval / avgInterval.
func (c *emaDynamic) ValuePer(interval time.Duration) float64 {
	return c.Value() * float64(interval) / float64(c.getDynamicInterval())
}

func (c *emaDynamic) LastValuePer(interval time.Duration) float64 {
	return c.valueAfter(0) * float64(interval) / float64(c.getDynamicInterval())
}

// Add adds a new value to the counter.
func (c *emaDynamic) Add(v float64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.startedAt.IsZero() {
		c.startedAt = getNow()
	}

	mul := 1.0
	newInterval := c.getDynamicInterval()

	if newInterval > c.lastInterval {
		if c.lastInterval != 0 {
			// mul normalises last value to the new increased interval
			mul = float64(newInterval) / float64(c.lastInterval)
		}
		c.lastInterval = newInterval
	}

	c.lastValue = c.value()*mul + v
	c.lastTime = getNow()
}

// Private methods

func (c *emaDynamic) value() float64 {
	return c.valueAfter(getNow().Sub(c.lastTime))
}

// calculates value decay
func (c *emaDynamic) valueAfter(elapsed time.Duration) float64 {
	if c.lastValue == 0 {
		return 0
	}

	return c.lastValue * math.Exp(-float64(elapsed)/float64(c.getDynamicInterval()))
}

func (c *emaDynamic) getDynamicInterval() time.Duration {
	elapsed := getNow().Sub(c.startedAt)
	if elapsed < c.avgInterval {
		if elapsed < c.minAvgInterval {
			return c.minAvgInterval
		}
		return elapsed
	}
	return c.avgInterval
}
