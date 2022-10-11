package lib

import (
	"time"
)

// RetryFn returns time to delay next call, and sets done to true when it was the last attempt
type RetryFn = func(attempt int) (delay time.Duration, done bool)

func NewLinearBackoff(delay time.Duration, maxAttempts *int, maxDelay *time.Duration) RetryFn {
	return func(attempt int) (wait time.Duration, done bool) {
		if maxAttempts != nil && attempt > *maxAttempts {
			return time.Duration(0), true
		}
		thisDelay := time.Duration(attempt) * delay
		if maxDelay != nil && thisDelay > *maxDelay {
			return *maxDelay, false
		}
		return thisDelay, false
	}
}

func NewExponentialBackoff() RetryFn {
	return nil
}
