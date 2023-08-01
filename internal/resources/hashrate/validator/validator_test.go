package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
)

type DestMock struct {
	vmask       string
	xnonce      string
	xnonce2size int
}

func (d *DestMock) GetVersionRolling() (versionRolling bool, versionRollingMask string) {
	return true, d.vmask
}
func (d *DestMock) GetExtraNonce() (extraNonce string, extraNonceSize int) {
	return d.xnonce, 8
}

func TestValidatorValidateUniqueShare(t *testing.T) {
	msg := GetTestMsg()
	dm := &DestMock{
		vmask:       msg.vmask,
		xnonce:      msg.xnonce,
		xnonce2size: msg.xnonce2size,
	}

	validator := NewValidator(dm, &lib.LoggerMock{})
	validator.AddNewJob(msg.notify, msg.diff)

	_, err := validator.ValidateAndAddShare(msg.submit1)
	require.NoError(t, err)

	_, err = validator.ValidateAndAddShare(msg.submit2)
	require.NoError(t, err)
}

func TestValidatorValidateDuplicateShare(t *testing.T) {
	msg := GetTestMsg()

	dm := &DestMock{
		vmask:       msg.vmask,
		xnonce:      msg.xnonce,
		xnonce2size: msg.xnonce2size,
	}

	validator := NewValidator(dm, &lib.LoggerMock{})
	validator.AddNewJob(msg.notify, msg.diff)

	_, err := validator.ValidateAndAddShare(msg.submit1)
	require.NoError(t, err)

	_, err = validator.ValidateAndAddShare(msg.submit1)
	require.ErrorIs(t, err, ErrDuplicateShare)
}
