package proxy

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
)

func TestReadCancellation(t *testing.T) {
	delay := 50 * time.Millisecond
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := NewConnection(client, &url.URL{}, 1*time.Minute, time.Now(), lib.NewTestLogger())

	go func() {
		// first and only write
		_, err := server.Write(append(stratumv1_message.NewMiningAuthorize(0, "0", "0").Serialize(), lib.CharNewLine))
		if err != nil {
			t.Error(err)
		}
	}()

	// read first message ok
	_, err := conn.Read(ctx)
	require.NoError(t, err)

	go func() {
		time.Sleep(delay)
		cancel()
	}()

	// read second message should block, and then be cancelled
	t1 := time.Now()
	_, err = conn.Read(ctx)

	require.ErrorIs(t, err, context.Canceled)
	require.GreaterOrEqual(t, time.Since(t1), delay)
}

func TestWriteCancellation(t *testing.T) {
	delay := 50 * time.Millisecond
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stratumClient := NewConnection(client, &url.URL{}, 1*time.Minute, time.Now(), lib.NewTestLogger())
	stratumServer := NewConnection(server, &url.URL{}, 1*time.Minute, time.Now(), lib.NewTestLogger())

	go func() {
		// first and only read
		_, err := stratumServer.Read(context.Background())
		if err != nil {
			t.Error(err)
		}
	}()

	// write first message ok
	err := stratumClient.Write(ctx, stratumv1_message.NewMiningAuthorize(0, "0", "0"))
	require.NoError(t, err)

	go func() {
		time.Sleep(delay)
		cancel()
	}()

	// write second message should block, and then be cancelled
	t1 := time.Now()
	err = stratumClient.Write(ctx, stratumv1_message.NewMiningAuthorize(1, "0", "0"))

	require.ErrorIs(t, err, context.Canceled)
	require.GreaterOrEqual(t, time.Since(t1), delay)
}

func TestConnTimeout(t *testing.T) {
	delay := 50 * time.Millisecond
	allowance := 50 * time.Millisecond
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	clientConn := NewConnection(client, &url.URL{}, delay, time.Now(), lib.NewTestLogger().Named("client"))
	serverConn := NewConnection(server, &url.URL{}, delay, time.Now(), lib.NewTestLogger().Named("server"))

	go func() {
		// try to read first message
		serverConn.Read(context.Background())
		// try to read second message, will fail due to timeout
		serverConn.Read(context.Background())
	}()

	// write first message ok
	err := clientConn.Write(context.Background(), stratumv1_message.NewMiningAuthorize(0, "0", "0"))
	require.NoError(t, err)

	time.Sleep(delay + allowance)

	// write second message, should fail due to timeout
	err = clientConn.Write(context.Background(), stratumv1_message.NewMiningAuthorize(0, "0", "0"))
	require.ErrorIs(t, err, os.ErrDeadlineExceeded)

	// try to read message, should fail as well due to timeout
	_, err = clientConn.Read(context.Background())
	require.ErrorIs(t, err, os.ErrDeadlineExceeded)

	// check if connection is closed
	// err = client.Close()
	// fmt.Println(err)

	_, err = clientConn.Read(context.Background())
	fmt.Println(err)

}
