package hashrate

import (
	"time"

	"github.com/gammazero/deque"
)

type measurement struct {
	timestamp time.Time
	value     float64
}

// Sma is an SMA (Simple Moving Average) counter.
type Sma struct {
	window time.Duration
	deque  *deque.Deque[measurement]
	sum    float64
	value  float64
	// mutex   sync.RWMutex
}

// NewEma creates a new Counter with the given window time
func NewSma(window time.Duration) *Sma {
	return &Sma{window: window, deque: deque.New[measurement](128, 128)}
}

// Add adds a new value to the counter.
func (c *Sma) Add(v float64) {
	c.deque.PushFront(measurement{value: v, timestamp: getNow()})
	c.adjustSum(+v)
}

// Value returns the current value of the counter.
func (c *Sma) Value() float64 {
	c.check()
	return c.value
}

func (c *Sma) ValuePer(t time.Duration) float64 {
	c.check()
	return c.value * float64(t)
}

func (c *Sma) check() {
	for {
		if c.deque.Len() == 0 {
			return
		}

		elem := c.deque.Back()
		if getNow().Sub(elem.timestamp) <= c.window {
			return
		}

		_ = c.deque.PopBack()
		c.adjustSum(-elem.value)
	}
}

func (c *Sma) adjustSum(v float64) {
	c.sum = c.sum + v
	c.value = c.sum / float64(c.window)
}

var _ Counter = new(Ema)