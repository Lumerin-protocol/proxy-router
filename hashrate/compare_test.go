package hashrate

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"text/tabwriter"
	"time"
)

func TestEmaV2(t *testing.T) {
	// t.Skip()
	diff := 500000
	interval := 5 * time.Minute
	submitInt := 60 * time.Second
	errMargin := 0.8
	MAX_OBSERVATIONS := 200

	rand.Seed(time.Now().UnixNano())

	startTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	nowTime = startTime

	emas := map[string]Counter{
		"ema":             NewEma(interval),
		"ema-dyn-30s":     NewEmaDynamic(interval, 30*time.Second),
		"ema-dyn-15s":     NewEmaDynamic(interval, 15*time.Second),
		"ema-primed":      NewEmaPrimed(interval, 10),
		"ema-primed-2.5m": NewEmaPrimed(interval/2, 10),
	}
	keys := []string{}
	for name := range emas {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	real := (float64(5*time.Minute) / float64(submitInt) * float64(diff)) / float64(5*time.Minute) * float64(time.Second)

	b := new(bytes.Buffer)

	w := tabwriter.NewWriter(b, 25, 2, 1, ' ', 0)

	fmt.Printf("real average value: %.2f\n", real)

	// table header
	fmt.Fprintf(w, "obs\telapsed\tdeltaT\t")
	for _, name := range keys {
		fmt.Fprintf(w, "%s\t", name)
	}
	fmt.Fprintf(w, "\n")

	var sleepTime time.Duration // time between additions

	// table rows
	for i := 0; i < MAX_OBSERVATIONS; i++ {
		fmt.Fprintf(w, "%d\t%s\t%s (%.2f)\t", i, getNow().Sub(startTime), sleepTime, delta(float64(sleepTime), float64(submitInt))) // observation number

		for _, name := range keys {
			ema := emas[name]
			ema.Add(float64(diff))
			val := ema.ValuePer(time.Second)
			fmt.Fprintf(w, "%.2f (%.2f)\t", val, delta(val, real))
		}
		fmt.Fprintf(w, "\n")

		_ = w.Flush()
		fmt.Printf("%s", b.String())

		b.Reset()

		errValue := time.Duration((rand.Float64() - 0.5) * 2 * errMargin * float64(submitInt))
		sleepTime = submitInt + errValue

		nowTime = nowTime.Add(sleepTime)
	}
}

func delta(act, exp float64) float64 {
	return (act - exp) / exp
}
