package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatorDiff(t *testing.T) {
	msg := GetTestMsg()

	diff, ok := ValidateDiff(msg.xnonce, uint(msg.xnonce2size), uint64(msg.diff), msg.vmask, msg.notify, msg.submit1)

	require.Truef(t, ok, "Result diff (%d) doesn't meet difficulty target (%.2f)", diff, msg.diff)
}

func TestValidatorDiffInvalidMsg(t *testing.T) {}

func TestValidatorDiffMalformedMsg(t *testing.T) {}
