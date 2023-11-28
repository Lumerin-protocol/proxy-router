package lib

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMutexTimeout(t *testing.T) {
	m := NewMutex()
	timeout := time.Millisecond * 40

	// test lock
	m.Lock()
	start := time.Now()
	err := m.LockTimeout(timeout)
	require.ErrorIsf(t, err, ErrTimeout, "locked mutex should timeout")
	require.InEpsilonf(t, timeout, time.Since(start), 0.1, "timeout should be close to %s", timeout)

	// test unlock
	m.Unlock()
	err = m.LockTimeout(0)
	require.NoErrorf(t, err, "unlocked mutex should not return error")

	// unlock of unlocked
	m.Unlock()
	err = m.LockTimeout(0)
	require.NoError(t, err, "unlock of unlocked mutex should not block")
}

func TestMutexCtx(t *testing.T) {
	m := NewMutex()
	timeout := time.Millisecond * 40
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// test lock
	m.Lock()
	start := time.Now()
	err := m.LockCtx(ctx)
	require.ErrorIsf(t, err, context.DeadlineExceeded, "locked mutex should timeout")
	require.InEpsilonf(t, timeout, time.Since(start), 0.1, "timeout should be close to %s", timeout)

	// test unlock
	m.Unlock()
	err = m.LockCtx(context.Background())
	require.NoErrorf(t, err, "unlocked mutex should not return error")

	// unlock of unlocked
	m.Unlock()
	err = m.LockCtx(context.Background())
	require.NoError(t, err, "unlock of unlocked mutex should not block")
}
