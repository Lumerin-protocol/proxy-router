package connectionmanager

import (
	"fmt"
	"testing"

	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

func TestListenOpen(t *testing.T) {

	_, e := Listen(12345)
	if e != nil {
		t.Fatalf(fmt.Sprintf(lumerinlib.FileLine()+"Listen() Test Failed: %s", e))
	}
}
