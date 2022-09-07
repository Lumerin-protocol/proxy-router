package hashrate

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestEma(t *testing.T) {
	counter := New(time.Second)
	counter.Add(10.0)
	require.LessOrEqual(t, counter.Value(),float64(10.0))
	val:=counter.ValuePer(time.Second)
	require.Less(t, val,float64(10))
	val=counter.LastValuePer(time.Second)
	require.LessOrEqual(t, val,float64(10))

	counter.Add(20.0)
	val=counter.Value()
	require.Greater(t, val,float64(29))
	require.Less(t, val,float64(30))

	val=counter.valueAfter(time.Second)
	require.Greater(t, val,float64(10))
	require.Less(t, val,float64(12))

}
