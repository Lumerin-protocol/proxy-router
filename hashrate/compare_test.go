package hashrate

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"
)

func TestEmaV2(t *testing.T) {
	t.Skip()
	diff := 50000
	interval := 2 * time.Minute
	ema := NewEma(interval)
	emaDyn := NewEmaDynamic(interval, 30*time.Second)
	emaPrimed := NewEmaPrimed(interval, 10)
	real := (float64(5*time.Minute) / float64(2*time.Second) * float64(diff)) / float64(5*time.Minute) * float64(time.Second)
	go func() {
		fmt.Printf("real value - %.2f\n", real)
		fmt.Printf("ema\tema imprv\temav2\temav3\n")
		for i := 0; ; i++ {
			ema.Add(float64(diff))
			emaDyn.Add(float64(diff))
			emaPrimed.Add(float64(diff))
			val1, val2, val3 := ema.ValuePer(time.Second), emaDyn.ValuePer(time.Second), emaPrimed.ValuePer(time.Second)
			fmt.Printf("%d\t%.2f\t%.2f\t%.2f\n", i, val1, val2, val3)
			fmt.Printf("\t%.2f\t%.2f\t%.2f\n\n", delta(val1, real), delta(val2, real), delta(val3, real))
			r, _ := rand.Int(rand.Reader, big.NewInt(4000))
			time.Sleep(time.Duration(r.Int64()) * time.Millisecond)
		}
	}()
	time.Sleep(time.Hour)
}

func delta(v1, v2 float64) float64 {
	return math.Abs(v1-v2) / v1
}
