package hashrate

import (
	"time"
)

type Counter interface {
	Add(v float64)
	Value() float64
	ValuePer(t time.Duration) float64
	Reset()
}

type HashrateFactory = func() *Hashrate
