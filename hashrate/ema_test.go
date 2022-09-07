package hashrate

import (
	"fmt"
	"testing"
	"time"
)

func TestEma(t *testing.T) {
	t.Skip()

	counter := New(5 * time.Hour)

	for i := 0; i < 40; i++ {
		counter.Add(10.0)
		time.Sleep(time.Second)
		fmt.Println(counter.LastValue(), counter.Value())
	}

	// The result about 60 (60 adds/minute)

	// The result about 60 (60 adds/minute)

}
