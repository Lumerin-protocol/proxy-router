package poolmock

import (
	"context"
	"testing"

	"gitlab.com/TitanInd/hashrouter/lib"
)

func TestPoolMock(t *testing.T) {
	t.SkipNow()
	poolMock := NewPoolMock(6666, lib.NewTestLogger())
	err := poolMock.Run(context.Background())
	if err != nil {
		t.Error(err)
	}
}
