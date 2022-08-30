package hashrate

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestHashrate(t *testing.T) {
	t.Skip()

	log, _ := zap.NewDevelopment()
	hashrate := NewHashrate(log.Sugar(),5*time.Minute)

	for i := 0; i < 100; i++ {
		hashrate.OnSubmit(10000)
		//fmt.Printf("Current Hashrate %d\n", hashrate.GetHashrate())
		time.Sleep(1 * time.Second)
	}

}
