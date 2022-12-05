package hashrate

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHashrate(t *testing.T) {
	hashrate := NewHashrate()

	for i := 0; i < 5; i++ {
		hashrate.OnSubmit(10000)
		fmt.Printf("Current Hashrate %d\n", hashrate.GetHashrateGHS())
		time.Sleep(100 * time.Millisecond)
	}

	require.Equal(t, hashrate.GetTotalHashes(), uint64(50000))
	require.InEpsilon(t, 712, hashrate.GetHashrateGHS(), 0.01)
}
