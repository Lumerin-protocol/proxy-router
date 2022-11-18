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

	// t.Logf("%d, %d", dq.Len(), dq.Cap())

	for i := 0; i < cap+1; i++ {
		dest, _ := lib.ParseDest(fmt.Sprintf("stratum+tcp://user:pwd@host.com:%d", i))
		time.Sleep(time.Second)
		history.Add(dest, "some-contract-id", nil)
	}

	history.Range(func(item HistoryItem) bool {
		t.Logf("item %s %s", item.Dest, item.Duration)
		return true
	})
}
