package fakeminerpool

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

func TestAddToWaitGroupAfterCallingWait(t *testing.T) {
	testErr := fmt.Errorf("test error")

	grp, _ := errgroup.WithContext(context.Background())
	grp.Go(func() error {
		time.Sleep(30 * time.Second)
		return nil
	})

	go func() {
		time.Sleep(3 * time.Second)
		grp.Go(func() error {
			t.Logf("new errgroup goroutine that is created after called wait")
			return testErr
		})
	}()

	err := grp.Wait()
	if !errors.Is(err, testErr) {
		t.Error("expected to get a testErr")
	}
}
