package contractmanager

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/TitanInd/hashrouter/hashrate"
)

func TestGlobalHashrate(t *testing.T) {
	// nowTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	threads := 100
	diff := int64(100000)
	loops := 100
	duration := 50 * time.Millisecond * time.Duration(loops)
	workerName := "kiki"

	hr := NewGlobalHashrate()

	cb := func(thread int) {
		for i := 0; i < loops; i++ {
			hr.OnSubmit(workerName, diff)
			time.Sleep(50 * time.Millisecond)
		}
	}

	for i := 0; i < threads; i++ {
		go cb(i)
	}

	time.Sleep(duration + time.Second)

	work := float64(threads*loops*int(diff)) / (9 * time.Minute).Seconds()
	expected := hashrate.HSToGHS(hashrate.JobSubmittedToHS(work))
	av, _ := hr.GetHashRateGHS(workerName)

	fmt.Printf("exp %d act %d", expected, av)
	assert.InEpsilon(t, expected, av, 0.01, "should be accurate")
}
