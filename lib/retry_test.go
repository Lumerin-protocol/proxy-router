package lib

import (
	"testing"
	"time"
)

func TestLinearBackoff(t *testing.T) {
	step := time.Second
	maxDelay := time.Second * 4
	maxAttempts := 5
	backoff := NewLinearBackoff(step, &maxAttempts, &maxDelay)
	for i := 0; ; i++ {
		delay, done := backoff(i)
		if done {
			break
		}
		t.Logf("delay %v attempt %d", delay.Seconds(), i)
		expected := time.Duration(i) * step
		if delay != expected && delay != maxDelay {
			t.Errorf("invalid delay: expected %d actual %d", expected, delay)
		}
		if delay > maxDelay {
			t.Errorf("delay is longer than maxDelay: expected less than %d actual %d", maxDelay, delay)
		}
		if i > maxAttempts {
			t.Errorf("more attempts than maxAttempts: expected less than %d actual %d", maxAttempts, i)
		}
	}
}
