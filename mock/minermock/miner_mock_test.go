package minermock

import (
	"context"
	"testing"

	"gitlab.com/TitanInd/hashrouter/lib"
)

// TestMinerMock used for manual mock miner run
func TestMinerMock(t *testing.T) {
	t.SkipNow()

	dest, _ := lib.ParseDest("tcp://shev8.local:anything123@0.0.0.0:5555")

	minerMock := NewMinerMock(dest, lib.NewTestLogger())
	minerMock.SetMinerGHS(1000)
	err := minerMock.Run(context.Background())
	if err != nil {
		t.Error(err)
	}
}
