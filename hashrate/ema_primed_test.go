package hashrate

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/TitanInd/hashrouter/lib"
)

func TestEmaPrimedShouldReach5PercentErr(t *testing.T) {
	avgInterval := time.Second
	iterations := 200
	targetErr := 0.1
	targetAvgObs := 40.0

	avgObs := testEmaMulti(iterations, func() Counter {
		return NewEmaPrimed(avgInterval, 10)
	}, targetErr)

	fmt.Printf("finised with average %.2f attempts", avgObs)
	require.NotZero(t, avgObs)
	require.Lessf(t, avgObs, targetAvgObs, "expected average observations (%.2f) to be less than (%d)", avgObs, targetAvgObs)
}

func testEmaMulti(iterations int, factory func() Counter, targetErr float64) float64 {
	maxObservations := 200

	totalObs := 0
	resCh := make(chan int)
	for i := 0; i < iterations; i++ {
		go func() {
			counter := factory()
			_, _, obs := testEma(maxObservations, counter, targetErr)
			resCh <- obs
		}()
	}

	for i := 0; i < iterations; i++ {
		val := <-resCh
		if val == 0 {
			return 0
		}
		totalObs += val
	}

	avgObs := float64(totalObs) / float64(iterations)
	return avgObs
}

func testEma(maxObservations int, counter Counter, targetErr float64) (float64, float64, int) {
	diff := 10000.0
	avgAddDelay := 30 * time.Millisecond
	expectedAvg := diff / float64(avgAddDelay) * float64(time.Second)
	actualAvg := 0.0
	for i := 1; i <= maxObservations; i++ {
		counter.Add(diff)
		actualAvg = counter.ValuePer(time.Second)
		fmt.Printf("%d  %.0f  %.0f  %.2f\n", i, actualAvg, expectedAvg, math.Abs(actualAvg-expectedAvg)/expectedAvg)
		if lib.AlmostEqual(expectedAvg, actualAvg, targetErr) {
			return expectedAvg, actualAvg, i
		}
		r, _ := rand.Int(rand.Reader, big.NewInt(int64(avgAddDelay)*2))
		time.Sleep(time.Duration(r.Int64()))
	}
	return expectedAvg, actualAvg, 0
}
