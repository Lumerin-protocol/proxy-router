package hashrate

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/TitanInd/hashrouter/lib"
)

func TestEmaPrimedShouldReach5PercentErr(t *testing.T) {
	avgInterval := 5 * time.Minute
	iterations := 200
	targetErr := 0.05
	targetAvgObs := 40

	avgObs := testEmaMulti(iterations, func() Counter {
		return NewEmaPrimed(avgInterval, 10)
	}, targetErr)

	fmt.Printf("finised with average %.2f attempts\n", avgObs)
	require.NotZero(t, avgObs)
	require.Lessf(t, avgObs, float64(targetAvgObs), "expected average observations (%.2f) to be less than (%d)", avgObs, targetAvgObs)
}

func testEmaMulti(iterations int, factory func() Counter, targetErr float64) float64 {
	maxObservations := 200
	nowTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	totalObs := 0
	for i := 0; i < iterations; i++ {
		counter := factory()
		_, _, obs := testEma(maxObservations, counter, targetErr)
		if obs == 0 {
			return 0
		}
		totalObs += obs
	}

	avgObs := float64(totalObs) / float64(iterations)
	return avgObs
}

func testEma(maxObservations int, counter Counter, targetErr float64) (float64, float64, int) {
	diff := 100000.0
	avgAddDelay := 15 * time.Second
	addDelayErr := 0.5

	expectedAvg := diff / float64(avgAddDelay) * float64(time.Second)
	actualAvg := 0.0

	for i := 1; i <= maxObservations; i++ {
		counter.Add(diff)
		actualAvg = counter.ValuePer(time.Second)

		if lib.AlmostEqual(expectedAvg, actualAvg, targetErr) {
			return expectedAvg, actualAvg, i
		}

		errValue := time.Duration((rand.Float64() - 0.5) * 2 * addDelayErr * float64(avgAddDelay))
		sleepTime := time.Duration(avgAddDelay + errValue)

		nowTime = nowTime.Add(sleepTime)
	}

	return expectedAvg, actualAvg, 0
}
