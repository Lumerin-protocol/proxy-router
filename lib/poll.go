package lib

import (
	"context"
	"time"
)

// Poll calls f function until it returns not an error or duration elapses. Context can be used to cancel
// execution prematurely. Optional interval parameter defines delay between function invocation
func Poll(ctx context.Context, dur time.Duration, f func() error, interval ...time.Duration) error {
	pollInterval := 1 * time.Second
	if len(interval) > 0 {
		pollInterval = interval[0]
	}

	for i := 0; ; i++ {
		err := f()
		if err == nil {
			return nil
		}

		elapsed := time.Duration(i) * pollInterval
		if err != nil && elapsed > dur {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
