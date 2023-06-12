package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"

	gi "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	i "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/interfaces"
	sm "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
)

// DestConn is a destination connection, a wrapper around StratumConnection,
// with destination specific state variables
type DestConn struct {
	// config
	workerName string
	destUrl    url.URL

	// state
	diff                float64
	hr                  gi.Hashrate
	isHandshakeComplete bool

	notifyMsgs     *lib.BoundStackMap[*sm.MiningNotify]
	autoReadSignal chan bool
	autoReadDone   chan struct{}

	resultHandlers      sync.Map // map[string]func(*stratumv1_message.MiningResult)
	resultHandlersErrCh chan error

	extraNonce     string
	extraNonceSize int

	versionRolling     bool
	versionRollingMask string

	// deps
	conn *StratumConnection
	log  gi.ILogger
}

const (
	NOTIFY_MSGS_CACHE_SIZE = 30
)

func NewDestConn(conn *StratumConnection, workerName string, log gi.ILogger) *DestConn {
	return &DestConn{
		workerName:     workerName,
		conn:           conn,
		log:            log,
		notifyMsgs:     lib.NewBoundStackMap[*sm.MiningNotify](NOTIFY_MSGS_CACHE_SIZE),
		autoReadSignal: make(chan bool),
	}
}

func ConnectDest(ctx context.Context, destURL *url.URL, log gi.ILogger) (*DestConn, error) {
	destLog := log.Named(fmt.Sprintf("[DST] %s@%s", destURL.User.Username(), destURL.Host))
	conn, err := Connect(destURL, CONNECTION_TIMEOUT, destLog)
	if err != nil {
		return nil, err
	}

	return NewDestConn(conn, "DEFAULT_WORKER", destLog), nil
}

func (c *DestConn) Run(ctx context.Context) error {
	var (
		subCtx    context.Context
		subCancel context.CancelFunc
		errCh     = make(chan error)
	)

	for {
		select {
		case sig := <-c.autoReadSignal:
			c.log.Warn("autoReadSignal received")
			c.autoReadDone = make(chan struct{})
			switch sig {
			case true:
				if subCtx != nil {
					c.log.Warn("autoReadSignal received while autoRead is already running")
				}
				subCtx, subCancel = context.WithCancel(ctx)
				go func() {
					c.log.Warn("autoRead started")

					errCh <- c.autoRead(subCtx)
					close(errCh)
				}()
			case false:
				if subCancel != nil {
					subCancel()
				}
			}
		case err := <-errCh:
			if subCancel != nil {
				subCancel()
			}
			subCtx, subCancel = nil, nil
			close(c.autoReadDone)
			if !errors.Is(err, context.Canceled) {
				return err
			}
		case err := <-c.resultHandlersErrCh:
			if subCancel != nil {
				subCancel()
			}
			<-errCh
			return err

		case <-ctx.Done():
			if subCancel != nil {
				subCancel()
			}
			err := <-errCh
			return err
		}
	}
}

// autoRead reads incoming jobs from the destination connection and
// caches them so dest will not close the connection
func (c *DestConn) autoRead(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_, err := c.Read(ctx)
		if err != nil {
			return err
		}
	}
}

func (c *DestConn) GetID() string {
	return c.conn.GetID()
}

func (c *DestConn) Read(ctx context.Context) (i.MiningMessageGeneric, error) {
	msg, err := c.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	return c.readInterceptor(msg)
}

func (c *DestConn) Write(ctx context.Context, msg i.MiningMessageGeneric) error {
	return c.conn.Write(ctx, msg)
}

func (c *DestConn) GetExtraNonce() (extraNonce string, extraNonceSize int) {
	return c.extraNonce, c.extraNonceSize
}

func (c *DestConn) SetExtraNonce(extraNonce string, extraNonceSize int) {
	c.extraNonce, c.extraNonceSize = extraNonce, extraNonceSize
}

func (c *DestConn) GetVersionRolling() (versionRolling bool, versionRollingMask string) {
	return c.versionRolling, c.versionRollingMask
}

func (c *DestConn) SetVersionRolling(versionRolling bool, versionRollingMask string) {
	c.versionRolling, c.versionRollingMask = versionRolling, versionRollingMask
}

// TODO: guard with mutex
func (c *DestConn) GetDiff() float64 {
	return c.diff
}

func (c *DestConn) GetHR() gi.Hashrate {
	return c.hr
}

func (c *DestConn) GetNotifyMsgJob(jobID string) (*sm.MiningNotify, bool) {
	return c.notifyMsgs.Get(jobID)
}

func (c *DestConn) AutoReadStart() {
	c.autoReadSignal <- true
}

func (c *DestConn) AutoReadStop() {
	c.autoReadSignal <- false
	<-c.autoReadDone
}

func (c *DestConn) readInterceptor(msg i.MiningMessageGeneric) (resMsg i.MiningMessageGeneric, err error) {
	switch typed := msg.(type) {
	case *sm.MiningNotify:
		c.notifyMsgs.Push(typed.GetJobID(), typed)
	case *sm.MiningSetDifficulty:
		c.diff = typed.GetDifficulty()
	case *sm.MiningSetExtranonce:
		c.extraNonce, c.extraNonceSize = typed.GetExtranonce()
	// TODO: handle set_version_mask, multiversion
	case *sm.MiningResult:
		handler, ok := c.resultHandlers.LoadAndDelete(typed.GetID())
		if ok {
			resMsg, err := handler.(ResultHandler)(typed)
			if err != nil {
				return nil, err
			}
			return resMsg, nil
		}
	}
	return msg, nil
}

// onceResult registers single time handler for the destination response with particular message ID,
// sets default timeout and does a cleanup when it expires
func (s *DestConn) onceResult(ctx context.Context, msgID int, handler ResultHandler) <-chan struct{} {
	done := make(chan struct{})

	ctx, cancel := context.WithTimeout(ctx, RESPONSE_TIMEOUT)
	didRun := false

	s.resultHandlers.Store(msgID, func(a *sm.MiningResult) (msg i.MiningMessageToPool, err error) {
		didRun = true
		defer cancel()
		defer close(done)
		return handler(a)
	})

	go func() {
		<-ctx.Done()
		s.resultHandlers.Delete(msgID)
		if !didRun {
			s.resultHandlersErrCh <- fmt.Errorf("pool response timeout")
		}
	}()

	return done
}
