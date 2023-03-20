package hashrate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSma(t *testing.T) {
	nowTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	sma := NewSma(15 * time.Minute)

	sma.Add(100_000)
	sleepNow(5 * time.Minute)
	sma.Add(100_000)
	sleepNow(5 * time.Minute)
	sma.Add(100_000)
	sleepNow(5 * time.Minute)

	av := sma.ValuePer(time.Minute)
	assert.InEpsilon(t, 20_000, av, 0.01, "should be accurate")
}

func TestSmaOutOfWindow(t *testing.T) {
	nowTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	sma := NewSma(15 * time.Minute)

	sma.Add(100_000)
	sleepNow(5 * time.Minute)
	sma.Add(100_000)
	sleepNow(5 * time.Minute)
	sma.Add(100_000)
	sleepNow(5 * time.Minute)
	sma.Add(100_000)
	sleepNow(5 * time.Minute)

	av := sma.ValuePer(time.Minute)
	assert.InEpsilon(t, 20_000, av, 0.01, "should be accurate")
}

func TestSmaZero(t *testing.T) {
	nowTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	sma := NewSma(15 * time.Minute)

	av := sma.ValuePer(time.Minute)
	assert.Equal(t, 0.0, av, "should be accurate")
}
