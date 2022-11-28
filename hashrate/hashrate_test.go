package hashrate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

func TestHashrate(t *testing.T) {
	log, _ := zap.NewDevelopment()
	hashrate := NewHashrate(log.Sugar())

	for i := 0; i < 5; i++ {
		hashrate.OnSubmit(10000)
		t.Logf("current Hashrate %d\n", hashrate.GetHashrateGHS())
		time.Sleep(100 * time.Millisecond)
	}

	require.Equal(t, hashrate.GetTotalHashes(), uint64(50000))
	require.InDelta(t, 712, hashrate.GetHashrateGHS(), 0.01)
}

func TestHashrateCustom(t *testing.T) {
	log, _ := zap.NewDevelopment()
	hashrate := NewHashrateCustom(log.Sugar(), 5*time.Minute)

	for i := 0; i < 5; i++ {
		hashrate.OnSubmit(10000)
		t.Logf("current Hashrate %d\n", hashrate.GetHashrateGHS())
		time.Sleep(100 * time.Millisecond)
	}

	custom5MinGHS, ok := hashrate.GetHashrateAvgGHSCustom(5 * time.Minute)
	if !ok {
		t.Fatalf("custom ema not found")
	}

	require.InDelta(t, 712, custom5MinGHS, 0.01)
}
