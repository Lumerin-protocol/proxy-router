package proxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"
	"time"

	gi "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
)

const (
	DIAL_TIMEOUT  = 10 * time.Second
	WRITE_TIMEOUT = 10 * time.Second

	READ_CLOSE_TIMEOUT  = 10 * time.Minute
	WRITE_CLOSE_TIMEOUT = 10 * time.Minute
)

type StratumConnection struct {
	// config
	id string

	// connTimeout      time.Duration
	// connection will automatically close if no read (write) operation is performed for this duration
	// the read/write operation will return
	connReadTimeout  time.Duration
	connWriteTimeout time.Duration

	address *url.URL

	// state
	connectedAt   time.Time
	reader        *bufio.Reader
	timeoutOnce   sync.Once
	readHappened  chan struct{}
	writeHappened chan struct{}
	closed        chan struct{}
	timersUpdated chan TimersUpdate

	// deps
	conn net.Conn
	log  gi.ILogger
}

type TimersUpdate struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// CreateConnection creates a new StratumConnection and starts background timer for its closure
func CreateConnection(conn net.Conn, address *url.URL, readTimeout, writeTimeout time.Duration, log gi.ILogger) *StratumConnection {
	c := &StratumConnection{
		id:               address.String(),
		conn:             conn,
		address:          address,
		connReadTimeout:  readTimeout,
		connWriteTimeout: writeTimeout,
		reader:           bufio.NewReader(conn),
		readHappened:     make(chan struct{}, 1),
		writeHappened:    make(chan struct{}, 1),
		closed:           make(chan struct{}),
		timersUpdated:    make(chan TimersUpdate),
		log:              log,
	}
	c.runTimeoutTimers()
	return c
}

// Connect connects to destination with
func Connect(address *url.URL, log gi.ILogger) (*StratumConnection, error) {
	conn, err := net.DialTimeout(address.Scheme, address.Host, DIAL_TIMEOUT)
	if err != nil {
		return nil, err
	}

	return CreateConnection(conn, address, READ_CLOSE_TIMEOUT, WRITE_CLOSE_TIMEOUT, log), nil
}

func (c *StratumConnection) runTimeoutTimers() {
	c.timeoutOnce.Do(func() {
		readTimer, writeTimer := time.NewTimer(c.connReadTimeout), time.NewTimer(c.connWriteTimeout)

		stopTimers := func() {
			if !readTimer.Stop() {
				<-readTimer.C
			}
			if !writeTimer.Stop() {
				<-writeTimer.C
			}
		}

		go func() {
			for {
				select {
				case <-readTimer.C:
					c.log.Debugf("connection %s read timeout", c.id)
					stopTimers()
					c.Close()
					return
				case <-writeTimer.C:
					c.log.Debugf("connection %s write timeout", c.id)
					stopTimers()
					c.Close()
					return
				case <-c.readHappened:
					if !readTimer.Stop() {
						<-readTimer.C
					}
					readTimer.Reset(c.connReadTimeout)
				case <-c.writeHappened:
					if !writeTimer.Stop() {
						<-writeTimer.C
					}
					writeTimer.Reset(c.connWriteTimeout)
				case <-c.closed:
					stopTimers()
					return
				case data := <-c.timersUpdated:
					stopTimers()
					readTimer.Reset(data.ReadTimeout)
					writeTimer.Reset(data.WriteTimeout)
				}
			}
		}()
	})
}

func (c *StratumConnection) SetCloseTimeout(readTimeout, writeTimeout time.Duration) {
	c.connReadTimeout, c.connWriteTimeout = readTimeout, writeTimeout
	c.timersUpdated <- TimersUpdate{ReadTimeout: readTimeout, WriteTimeout: writeTimeout}
}

func (c *StratumConnection) GetID() string {
	return c.id
}

func (c *StratumConnection) Read(ctx context.Context) (interfaces.MiningMessageGeneric, error) {
	c.runTimeoutTimers()
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
		err := c.conn.SetReadDeadline(time.Now().Add(c.connReadTimeout))
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
	c.runTimeoutTimers()
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

	err = c.conn.SetWriteDeadline(time.Now().Add(c.connWriteTimeout))
	if err != nil {
		return err
	}
	return nil
}

func (c *StratumConnection) Close() error {
	defer close(c.closed)
	return c.conn.Close()
}

func (c *StratumConnection) GetConnectedAt() time.Time {
	return c.connectedAt
}
