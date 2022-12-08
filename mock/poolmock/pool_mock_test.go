package poolmock

import (
	"context"
	"testing"

	"gitlab.com/TitanInd/hashrouter/lib"
)

// TestPoolMock used for manual mock pool run
func TestPoolMock(t *testing.T) {
	t.SkipNow()
	ctx := context.Background()
	poolMock := NewPoolMock(6666, lib.NewTestLogger())
	err := poolMock.Connect(ctx)
	if err != nil {
		t.Error(err)
	}
	err = poolMock.Run(ctx)
	if err != nil {
		t.Error(err)
	}
}
