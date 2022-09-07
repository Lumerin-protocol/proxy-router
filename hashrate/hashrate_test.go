package hashrate

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestHashrate(t *testing.T) {
	log, _ := zap.NewDevelopment()
	hashrate := NewHashrate(log.Sugar(),time.Second)

	for i := 0; i < 5; i++ {
		hashrate.OnSubmit(10000)
		fmt.Printf("Current Hashrate %d\n", hashrate.getHashrateHS())
		time.Sleep(1 * time.Second)
	}
	require.Equal(t, hashrate.GetTotalHashes(),uint64(50000))
}
