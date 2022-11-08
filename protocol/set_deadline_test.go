package protocol

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"
)

func TestSetReadDeadline(t *testing.T) {
	server, client := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Do some stuff
	LOOP:
		for {
			bytes := make([]byte, 10)
			_, err := server.Read(bytes)
			if err != nil {
				t.Log(err)
			}
			t.Log(string(bytes))
			select {
			case <-ctx.Done():
				break LOOP
			default:
			}
		}
		server.Close()
	}()

	// Do some stuff
	_, err := client.Write([]byte("before"))
	if err != nil {
		t.Log(err)
	}
	err = client.SetWriteDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		t.Log(err)
	}
	time.Sleep(2 * time.Second)
	_, err = client.Write([]byte("after"))

	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Error("should be os.ErrDeadlineExceeded error")
	}

	client.Close()
	cancel()
}
