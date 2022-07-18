package connections

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestParentCtxCancel(t *testing.T) {
	t.Skip()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(10 * time.Second)
		cancel()
	}()

	err := testFn(ctx)
	fmt.Printf("resulting error %#+v %T\n", err, err)
}

func testFn(ctx context.Context) error {
	for {
		// if outer context is canceled then stop processing and return error
		select {
		case <-ctx.Done():
			return errors.New("cancelled from parent")
		default:
		}
		err := run(ctx)
		if !errors.Is(err, context.Canceled) {
			return err
		}
		fmt.Printf("Cancelled from child... restart")
	}
}

func run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-time.After(5 * time.Second)
		cancel()
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		<-time.After(2 * time.Second)
		fmt.Printf("working...\n")
	}

}

func TestPendingGoroutine(t *testing.T) {
	go func() {
		go func() {
			for {
				time.Sleep(1 * time.Second)
				fmt.Println("inner is working")
			}
		}()

		<-time.After(5 * time.Second)
		fmt.Println("outer exited")
	}()

	<-time.After(15 * time.Second)
}
