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

	e.Subscribe(context.Background(), TestEventName, func(val interface{}) {
		ch <- val
	})
	e.Publish(TestEventName, "kiki")

	msg := <-ch
	t.Logf("Message: %+v", msg)
}

func TestSubscribeNotBlocking(t *testing.T) {
	e := NewEventBus()

	ch1 := make(chan DataEvent)
	ch2 := make(chan DataEvent)

	e.Subscribe(context.Background(), TestEventName, func(val interface{}) {
		ch1 <- val
	})
	e.Subscribe(context.Background(), TestEventName, func(val interface{}) {
		ch2 <- val
	})

	e.Publish(TestEventName, "kiki")

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

	err := gr.Wait()

	if err != nil {
		t.Fail()
	}

	if ch2Time >= ch1Time {
		t.Fail()
	}
}

func TestSubscribeCancel(t *testing.T) {
	e := NewEventBus()
	ch := make(chan DataEvent)

	ctx, cancel := context.WithCancel(context.Background())

	e.Subscribe(ctx, TestEventName, func(val interface{}) {
		ch <- val
	})

	e.Publish(TestEventName, "first-message")

	// wait for consumer to pick up first message
	time.Sleep(100 * time.Millisecond)

	select {
	case <-ch:
	default:
		t.Fatalf("First message should be reveived")
	}

	cancel()
	// make sure it is cancelled
	time.Sleep(100 * time.Millisecond)

	e.Publish(TestEventName, "second-message")

	select {
	case <-ch:
		t.Fatalf("Second message shouldn't be reveived")
	default:
	}
}
