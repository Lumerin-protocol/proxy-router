package hashrate

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEmaShouldReach5PercentErr(t *testing.T) {
	avgInterval := 5 * time.Minute
	iterations := 200
	targetErr := 0.05
	targetAvgObs := 55

	avgObs := testEmaMulti(iterations, func() Counter {
		return NewEma(avgInterval)
	}, targetErr)

	fmt.Printf("finised with average %.2f attempts\n", avgObs)
	require.NotZero(t, avgObs)
	require.Lessf(t, avgObs, float64(targetAvgObs), "expected average observations (%.2f) to be less than (%d)", avgObs, targetAvgObs)
}
