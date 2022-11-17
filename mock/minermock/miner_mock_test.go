package minermock

import (
	"testing"

	"gitlab.com/TitanInd/hashrouter/lib"
)

func TestPoolConnection(t *testing.T) {
	dest, _ := lib.ParseDest("tcp://shev8.local:anything123@0.0.0.0:5555")
	minerMock := NewMinerMock(dest)
	err := minerMock.Run()
	if err != nil {
		t.Error(err)
	}
}
