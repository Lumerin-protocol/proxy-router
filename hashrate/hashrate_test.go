package hashrate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHashrate(t *testing.T) {
	nowTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	hashrate := NewHashrate()
	diff := 10000
	submitDelay := 5 * time.Second
	avgDuration := 5 * time.Minute
	observations := int(avgDuration/submitDelay) * 4

	expected := float64(diff*observations) / float64(observations*int(submitDelay)) * float64(time.Second)
	expectedGHS := hashrate.averageSubmitDiffToGHS(expected)
	expectedTotalHashes := observations * diff

	for i := 0; i < observations; i++ {
		hashrate.OnSubmit(int64(diff))
		// fmt.Printf("Current Time %s Hashrate %d\n", nowTime, hashrate.GetHashrateGHS())
		nowTime = nowTime.Add(submitDelay)
	}

	require.Equal(t, hashrate.GetTotalWork(), uint64(expectedTotalHashes))
	require.InEpsilon(t, expectedGHS, hashrate.GetHashrate5minAvgGHS(), 0.05)
}
