package validator

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShareEncode(t *testing.T) {
	submit := GetTestMsg().submit1

	bytes := HashShare("", submit.GetExtraNonce2(), submit.GetNtime(), submit.GetNonce(), submit.GetVmask())
	actual := hex.EncodeToString(bytes[:])
	expected := "00000000" + submit.GetExtraNonce2() + submit.GetNtime() + submit.GetNonce() + submit.GetVmask()

	require.Equal(t, expected, actual)
}

func TestMiningJob(t *testing.T) {
	msg := GetTestMsg()

	job := NewMiningJob(msg.notify, 8096)
	isDuplicate := job.CheckDuplicateAndAddShare(msg.submit1)
	require.False(t, isDuplicate)

	isDuplicate = job.CheckDuplicateAndAddShare(msg.submit1)
	require.True(t, isDuplicate)

	isDuplicate = job.CheckDuplicateAndAddShare(msg.submit2)
	require.False(t, isDuplicate)
}
