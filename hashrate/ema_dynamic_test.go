package hashrate

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEmaDynamicShouldReach5PercentErr(t *testing.T) {
	avgInterval := time.Second
	iterations := 200
	targetErr := 0.1
	targetAvgObs := 30.0

	avgObs := testEmaMulti(iterations, func() Counter {
		return NewEmaDynamic(avgInterval, 300*time.Millisecond)
	}, targetErr)

	fmt.Printf("finised with average %.2f attempts", avgObs)
	require.NotZero(t, avgObs)
	require.Lessf(t, avgObs, targetAvgObs, "expected average observations (%.2f) to be less than (%d)", avgObs, targetAvgObs)
}
