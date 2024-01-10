package validator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
)

func TestValidatorValidateUniqueShare(t *testing.T) {
	msg := GetTestMsg()

	validator := NewValidator(time.Minute, &lib.LoggerMock{})
	validator.SetVersionRollingMask(msg.vmask)
	validator.AddNewJob(msg.notify, msg.diff, msg.xnonce, msg.xnonce2size)

	_, err := validator.ValidateAndAddShare(msg.submit1)
	require.NoError(t, err)

	_, err = validator.ValidateAndAddShare(msg.submit2)
	require.NoError(t, err)
}

func TestValidatorValidateDuplicateShare(t *testing.T) {
	msg := GetTestMsg()

	validator := NewValidator(time.Minute, &lib.LoggerMock{})
	validator.SetVersionRollingMask(msg.vmask)
	validator.AddNewJob(msg.notify, msg.diff, msg.xnonce, msg.xnonce2size)

	_, err := validator.ValidateAndAddShare(msg.submit1)
	require.NoError(t, err)

	_, err = validator.ValidateAndAddShare(msg.submit1)
	require.ErrorIs(t, err, ErrDuplicateShare)
}

func TestValidatorExpiredJobs(t *testing.T) {
	msg := GetTestMsg()
	timeout := 100 * time.Millisecond

	validator := NewValidator(timeout, &lib.LoggerMock{})
	validator.SetVersionRollingMask(msg.vmask)
	validator.AddNewJob(msg.notify, msg.diff, msg.xnonce, msg.xnonce2size)

	_, err := validator.ValidateAndAddShare(msg.submit1)
	require.NoError(t, err)

	validator.ScheduleCleanJobs()

	time.Sleep(2 * timeout)

	_, err = validator.ValidateAndAddShare(msg.submit1)
	require.ErrorIs(t, err, ErrJobNotFound)
}
