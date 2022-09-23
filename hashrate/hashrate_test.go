package hashrate

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

func TestHashrate(t *testing.T) {
	log, _ := zap.NewDevelopment()
	hashrate := NewHashrate(log.Sugar(), time.Second)

	for i := 0; i < 5; i++ {
		hashrate.OnSubmit(10000)
		fmt.Printf("Current Hashrate %d\n", hashrate.GetHashrateGHS())
		time.Sleep(100 * time.Millisecond)
	}

	require.Equal(t, hashrate.GetTotalHashes(), uint64(50000))
	require.InDelta(t, 712, hashrate.GetHashrateGHS(), 0.01)
}
