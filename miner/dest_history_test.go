package miner

import (
	"fmt"
	"testing"
	"time"

	"gitlab.com/TitanInd/hashrouter/lib"
)

func TestHistory(t *testing.T) {
	cap := 8
	history := NewDestHistory(cap)

	for i := 0; i < cap+1; i++ {
		dest, _ := lib.ParseDest(fmt.Sprintf("stratum+tcp://user:pwd@host.com:%d", i))
		time.Sleep(time.Millisecond)
		history.Add(dest, "some-contract-id", nil)
	}

	count := 0
	history.Range(func(item HistoryItem) bool {
		count++
		return true
	})

	if count == 0 || count > cap {
		t.Fatalf("invalid history capacity, expected %d actual %d", cap, count)
	}
}
