package minermock

import (
	"context"
	"testing"

	"gitlab.com/TitanInd/hashrouter/lib"
)

func TestPoolConnection(t *testing.T) {
	t.SkipNow()

	dest, _ := lib.ParseDest("tcp://shev8.local:anything123@0.0.0.0:5555")
	minerMock := NewMinerMock(dest)
	err := minerMock.Run(context.Background())
	if err != nil {
		t.Error(err)
	}
}
