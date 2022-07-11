package eventbus

import (
	"context"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

// Simple publish subscribe
func TestEventBus(t *testing.T) {
	e := NewEventBus()
	ch := make(chan DataEvent)

	e.Subscribe("test", func(val interface{}) {
		ch <- val
	})
	e.Publish("test", "kiki")

	msg := <-ch
	t.Logf("Message: %+v", msg)
}

//
func TestBlockingRead(t *testing.T) {
	e := NewEventBus()

	ch1 := make(chan DataEvent)
	ch2 := make(chan DataEvent)

	e.Subscribe("test", func(val interface{}) {
		ch1 <- val
	})
	e.Subscribe("test", func(val interface{}) {
		ch2 <- val
	})

	e.Publish("test", "kiki")

	gr, _ := errgroup.WithContext(context.Background())

	startTime := time.Now()
	var ch1Time, ch2Time time.Duration

	gr.Go(func() error {
		<-time.After(100 * time.Millisecond)
		t.Logf("Received from channel 1 %s", <-ch1)
		ch1Time = time.Since(startTime)
		return nil
	})

	gr.Go(func() error {
		t.Logf("Received from channel 2 %s", <-ch2)
		ch2Time = time.Since(startTime)
		return nil
	})

	gr.Wait()

	if ch2Time >= ch1Time {
		t.Fail()
	}
}
