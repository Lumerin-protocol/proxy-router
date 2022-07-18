package connections

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"gitlab.com/TitanInd/hashrouter/lib"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	destTimeout = 5 * time.Second
)

var (
	ErrProxyWrite = errors.New("proxy-write-err")
	ErrProxyRead  = errors.New("proxy-read-err")
)

type ProxyConn struct {
	poolAddr  string
	minerConn net.Conn
	protocol  MessageHandler
	log       *zap.SugaredLogger

	PoolConn net.Conn // connection to the current pool

	cancel   context.CancelFunc // used for pause: stops proxying but doesn't return control to parent; can be resumed
	done     chan interface{}   // new message means that proxying has stopped
	resumeCh chan interface{}   // new message means that proxying should continue
}

func GetConnID(minerConnRemoteAddr string, poolConnRemoteAddr string) string {
	return fmt.Sprintf("%s->%s", minerConnRemoteAddr, poolConnRemoteAddr)
}

func NewProxyConn(poolAddr string, minerConn net.Conn, log *zap.SugaredLogger) *ProxyConn {
	return &ProxyConn{
		poolAddr:  poolAddr,
		minerConn: minerConn,
		log:       log,
		done:      make(chan interface{}),
		resumeCh:  make(chan interface{}),
	}
}

func (c *ProxyConn) SetHandler(pc MessageHandler) {
	c.protocol = pc
}

func (c *ProxyConn) ID() string {
	return GetConnID(c.minerConn.RemoteAddr().String(), c.PoolConn.RemoteAddr().String())
}

func (c *ProxyConn) DialDest() error {
	poolConn, err := net.DialTimeout("tcp", c.poolAddr, destTimeout)
	if err != nil {
		return fmt.Errorf("cannot connect to pool %s %w", c.poolAddr, err)
	}
	c.PoolConn = poolConn
	return nil
}

func (c *ProxyConn) ClosePoolConn() error {
	err := c.PoolConn.Close()
	if err != nil {
		c.log.Errorw("pool connection close error", "error", err)
	}

	// TODO fix error and check if connection closed
	return nil
}

func (c *ProxyConn) Run(ctx context.Context) error {
	for {
		// if parent context is canceled then stop processing and return error
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := c.run(ctx)
		// signalizes that proxying stopped
		c.done <- struct{}{}

		// context can be cancelled from either parent
		// context or child context. Parent cancellation
		// is handled above. Child context cancellation
		// used for pause, so it is ignored
		if !errors.Is(err, context.Canceled) {
			c.log.Errorf("proxy_conn error %+v", err)
			return err
		}

		c.log.Debugf("proxy_conn was paused")
		// wait for resume signal
		<-c.resumeCh
		c.log.Debugf("proxy_conn was resumed")
	}
}

func (c *ProxyConn) run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.done = make(chan interface{})

	if c.PoolConn == nil {
		return fmt.Errorf("connect() has not been called")
	}

	minerMsgCh := make(chan []byte)
	poolMsgCh := make(chan []byte)

	group, subCtx := errgroup.WithContext(ctx)

	// Proxying miner -> pool messages
	group.Go(func() error {
		return c.proxyReader(subCtx, c.minerConn, minerMsgCh)
	})

	// Proxying pool -> miner messages
	group.Go(func() error {
		return c.proxyReader(subCtx, c.PoolConn, poolMsgCh)
	})

	// Transforming and sending response
	group.Go(func() error {
		return c.proxyWriter(subCtx, minerMsgCh, poolMsgCh)
	})

	// On any error
	err := group.Wait()

	if err := c.ClosePoolConn(); err != nil {
		c.log.Errorw("error during closing pool conn", "error", err)
	}

	return err
}

func (c *ProxyConn) PauseProxy() {
	c.cancel()
	<-c.done
}

func (c *ProxyConn) ResumeProxy() {
	c.resumeCh <- true
}

func (c *ProxyConn) ChangePool(newPoolAddr string) error {
	newPoolConn, err := net.DialTimeout("tcp", newPoolAddr, destTimeout)
	if err != nil {
		return fmt.Errorf("cannot connect to pool %s %w", newPoolAddr, err)
	}
	c.log.Debugf("successfully dialed new pool %s", newPoolAddr)

	c.PauseProxy()

	c.PoolConn = newPoolConn
	c.poolAddr = newPoolAddr

	c.ResumeProxy()

	return nil
}

func (c *ProxyConn) proxyReader(ctx context.Context, sourceConn net.Conn, msgCh chan<- []byte) error {
	sourceReader := bufio.NewReader(sourceConn)
	resCh := make(chan []byte)
	errCh := make(chan error)
	go func() {
		for {
			//TODO: readbytes is blocking and keeps reading even after parent exits
			// figure out a way to unblock it
			msg, err := sourceReader.ReadBytes('\n')
			if err != nil {
				errCh <- lib.WrapError(ErrProxyRead, err)
				return
			}

			// trim the newline char at the end of the line
			msg = msg[:len(msg)-1]

			if len(msg) <= 0 {
				continue
			}

			resCh <- msg
		}
	}()

	for {
		select {
		case chunk := <-resCh:
			msgCh <- chunk
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *ProxyConn) proxyWriter(ctx context.Context, minerMsgCh <-chan []byte, poolMsgCh <-chan []byte) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case msg := <-minerMsgCh:
			msg = c.protocol.MinerMessageHandler(ctx, msg)
			if msg == nil {
				continue
			}
			err := c.WriteToPool(ctx, msg)
			if err != nil {
				return err
			}

		case msg := <-poolMsgCh:
			msg = c.protocol.PoolMessageHandler(ctx, msg)
			if msg == nil {
				continue
			}
			err := c.WriteToMiner(ctx, msg)
			if err != nil {
				return err
			}
		}
	}
}

func (c *ProxyConn) write(ctx context.Context, destConn net.Conn, msg []byte) error {
	msg = append(msg, []byte("\n")...)
	_, err := destConn.Write(msg)
	if err != nil {
		return lib.WrapError(ErrProxyWrite, err)
	}

	return nil
}

func (c *ProxyConn) WriteToMiner(ctx context.Context, msg []byte) error {
	return c.write(ctx, c.minerConn, msg)
}

func (c *ProxyConn) WriteToPool(ctx context.Context, msg []byte) error {
	return c.write(ctx, c.PoolConn, msg)
}
