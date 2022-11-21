package poolmock

import (
	"context"
	"testing"
)

func TestPoolMock(t *testing.T) {
	t.SkipNow()
	poolMock := NewPoolMock(6666)
	err := poolMock.Run(context.Background())
	if err != nil {
		t.Error(err)
	}
}
