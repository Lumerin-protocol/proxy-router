package protocol

import (
	"net"
	"testing"
	"time"

	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

func TestGetDestNoConnection(t *testing.T) {
	connPool := NewStratumV1PoolPool(lib.NewTestLogger(), 10*time.Minute, false)
	if connPool.GetDest() != nil {
		t.Fatalf("should return nil if no connection was established")
	}
}

func TestGetDestAfterConnection(t *testing.T) {
	log := lib.NewTestLogger()
	connTimeout := 10 * time.Minute
	connPool := NewStratumV1PoolPool(log, connTimeout, false)

	_, client := net.Pipe()
	dest, _ := lib.ParseDest("//test:test@0.0.0.0:0")
	confMsg, _ := stratumv1_message.ParseMiningConfigure([]byte(`{"method": "mining.configure","id": 1,"params": [["minimum-difficulty", "version-rolling"],{"minimum-difficulty.value": 2048, "version-rolling.mask": "1fffe000", "version-rolling.min-bit-count": 2}]}`))
	connPool.conn = NewStratumV1Pool(client, log, dest, confMsg, connTimeout, false)

	if connPool.GetDest() == nil {
		t.Fatalf("should return dest if connection was established")
	}
}

func TestGetDestAfterConnectionClosed(t *testing.T) {
	log := lib.NewTestLogger()
	connTimeout := 10 * time.Minute
	connPool := NewStratumV1PoolPool(log, connTimeout, false)

	_, client := net.Pipe()
	dest, _ := lib.ParseDest("//test:test@0.0.0.0:0")
	confMsg, _ := stratumv1_message.ParseMiningConfigure([]byte(`{"method": "mining.configure","id": 1,"params": [["minimum-difficulty", "version-rolling"],{"minimum-difficulty.value": 2048, "version-rolling.mask": "1fffe000", "version-rolling.min-bit-count": 2}]}`))
	connPool.conn = NewStratumV1Pool(client, log, dest, confMsg, connTimeout, false)

	err := connPool.Close()
	if err != nil {
		t.Fatalf("should close connection without error")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("should not panic after connection closed")
		}
	}()

	connPool.GetDest()
}