package proxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	globalInterfaces "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
)

const (
	DIAL_TIMEOUT  = 10 * time.Second
	WRITE_TIMEOUT = 10 * time.Second
)

type StratumConnection struct {
	// config
	id          string
	connTimeout time.Duration
	address     *url.URL

	// state
	connectedAt time.Time
	reader      *bufio.Reader

	// deps
	conn net.Conn
	log  globalInterfaces.ILogger
}

func NewConnection(conn net.Conn, address *url.URL, connTimeout time.Duration, connectedAt time.Time, log globalInterfaces.ILogger) *StratumConnection {
	if connTimeout != 0 {
		conn.SetDeadline(time.Now().Add(connTimeout))
	}
	return &StratumConnection{
		conn:        conn,
		reader:      bufio.NewReader(conn),
		address:     address,
		connTimeout: connTimeout,
		connectedAt: connectedAt,
		log:         log,
	}
}

func Connect(address *url.URL, connTimeout time.Duration, log globalInterfaces.ILogger) (*StratumConnection, error) {
	conn, err := net.DialTimeout(address.Scheme, address.Host, DIAL_TIMEOUT)
	if err != nil {
		return nil, err
	}

	return NewConnection(conn, address, connTimeout, time.Now(), log), nil
}

func (c *StratumConnection) GetID() string {
	return c.id
}

func (c *StratumConnection) Read(ctx context.Context) (interfaces.MiningMessageGeneric, error) {
	doneCh := make(chan struct{})
	defer close(doneCh)

	// cancellation via context is implemented using SetReadDeadline,
	// which unblocks read operation causing it to return os.ErrDeadlineExceeded
	// TODO: consider implementing it in separate goroutine instead of a goroutine per read
	go func() {
		select {
		case <-ctx.Done():
			err := c.conn.SetReadDeadline(time.Now())
			if err != nil {
				// may return ErrNetClosing if fd is already closed
				c.log.Warnf("err during setting read deadline: %s", err)
				return
			}
		case <-doneCh:
			return
		}
	}()

	for {
		err := c.conn.SetReadDeadline(time.Now().Add(c.connTimeout))
		if err != nil {
			return nil, err
		}

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		line, isPrefix, err := c.reader.ReadLine()

		if isPrefix {
			return nil, fmt.Errorf("line is too long for the buffer used: %s", string(line))
		}

		if err != nil {
			// if read was cancelled via context return context error, not deadline exceeded
			if ctx.Err() != nil && errors.Is(err, os.ErrDeadlineExceeded) {
				return nil, ctx.Err()
			}
			return nil, err
		}

		c.log.Debugf("<= %s", string(line))

		m, err := stratumv1_message.ParseStratumMessage(line)

		if err != nil {
			c.log.Errorf("unknown stratum message: %s", string(line))
			continue
		}

		return m, nil
	}
}

// Write writes message to the connection. Safe for concurrent use, cause underlying TCPConn is thread-safe
func (c *StratumConnection) Write(ctx context.Context, msg interfaces.MiningMessageGeneric) error {
	if msg == nil {
		return fmt.Errorf("nil message write attempt")
	}

	b := append(msg.Serialize(), lib.CharNewLine)

	doneCh := make(chan struct{})
	defer close(doneCh)

	ctx, cancel := context.WithTimeout(ctx, WRITE_TIMEOUT)

	// cancellation via context is implemented using SetReadDeadline,
	// which unblocks read operation causing it to return os.ErrDeadlineExceeded
	// TODO: consider implementing it in separate goroutine instead of a goroutine per read
	go func() {
		select {
		case <-ctx.Done():
			err := c.conn.SetWriteDeadline(time.Now())
			if err != nil {
				// may return ErrNetClosing if fd is already closed
				c.log.Warnf("err during setting write deadline: %s", err)
				return
			}
		case <-doneCh:
			cancel()
			return
		}
	}()

	_, err := c.conn.Write(b)

	if err != nil {
		// if read was cancelled via context return context error, not deadline exceeded
		if ctx.Err() != nil && errors.Is(err, os.ErrDeadlineExceeded) {
			return ctx.Err()
		}
		return err
	}

	c.log.Debugf("=> %s", string(msg.Serialize()))

	err = c.conn.SetWriteDeadline(time.Now().Add(c.connTimeout))
	if err != nil {
		return err
	}
	return nil
}

func (c *StratumConnection) Close() error {
	return c.conn.Close()
}
