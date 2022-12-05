package hashrate

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEma(t *testing.T) {
	counter := NewEma(time.Second)
	counter.Add(10.0)
	require.LessOrEqual(t, counter.Value(), float64(10.0))
	val := counter.ValuePer(time.Second)
	require.Less(t, val, float64(10))
	val = counter.LastValuePer(time.Second)
	require.LessOrEqual(t, val, float64(10))

	counter.Add(20.0)
	val = counter.Value()
	require.Greater(t, val, float64(29))
	require.Less(t, val, float64(30))

	val = counter.valueAfter(time.Second)
	require.Greater(t, val, float64(10))
	require.Less(t, val, float64(12))
}

func TestEmaShouldReach5PercentErr(t *testing.T) {
	avgInterval := time.Second
	iterations := 200
	targetErr := 0.1
	targetAvgObs := 80

	avgObs := testEmaMulti(iterations, func() Counter {
		return NewEma(avgInterval)
	}, targetErr)

	fmt.Printf("finised with average %.2f attempts", avgObs)
	require.NotZero(t, avgObs)
	require.Lessf(t, avgObs, float64(targetAvgObs), "expected average observations (%.2f) to be less than (%d)", avgObs, targetAvgObs)
}
