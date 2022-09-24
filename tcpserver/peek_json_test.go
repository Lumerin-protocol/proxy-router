package tcpserver

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestPeekJSON(t *testing.T) {
	server, client := net.Pipe()

	testJSON := []byte(`{"ID":"abc"}`)
	chunk1, chunk2 := testJSON[:6], testJSON[6:]

	go func() {
		_, _ = client.Write(chunk1)
		time.Sleep(time.Second)
		_, _ = client.Write(chunk2)
	}()

	msg, err := PeekJSON(NewBufferedConn(server))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(msg, testJSON) {
		t.Fatalf("expected %s actual %s", string(testJSON), string(msg))
	}
}
