package connections

import (
	"bufio"
	"context"
	"fmt"
	"net"

	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/protocol"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Writer interface {
	Write(msg []byte) (n int, err error)
}

type ProxyTransformer interface {
	ProcessMiningMessage(ctx context.Context, msg []byte, pс protocol.Connection) []byte
	ProcessPoolMessage(ctx context.Context, msg []byte, pс protocol.Connection) []byte
}

type ProxyConn struct {
	poolAddr    string
	minerConn   net.Conn
	transformer ProxyTransformer
	log         *zap.SugaredLogger

	done     chan interface{}
	cancel   context.CancelFunc
	PoolConn net.Conn
}

func GetConnID(minerConnRemoteAddr string, poolConnRemoteAddr string) string {
	return fmt.Sprintf("%s->%s", minerConnRemoteAddr, poolConnRemoteAddr)
}

func NewProxyConn(poolAddr string, minerConn net.Conn, log *zap.SugaredLogger, pc *protocol.StratumV1) *ProxyConn {
	return &ProxyConn{
		poolAddr:    poolAddr,
		minerConn:   minerConn,
		log:         log,
		transformer: pc,
	}
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

func (c *ProxyConn) Run(ctx context.Context) error {
	subCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.done = make(chan interface{})

	if c.PoolConn == nil {
		return fmt.Errorf("connect() has not been called")
	}

	minerMsgCh := make(chan []byte)
	poolMsgCh := make(chan []byte)

	group, subCtx := errgroup.WithContext(subCtx)

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
	c.PoolConn.Close()
	c.log.Warn("Proxy error. Pool connection closed", err)
	close(c.done)
	return err
}

func (c *ProxyConn) Stop() {
	c.cancel()
	<-c.done
}

func (c *ProxyConn) ChangePool(addr string) error {
	poolConn, err := net.DialTimeout("tcp", addr, destTimeout)
	if err != nil {
		return fmt.Errorf("cannot connect to pool %s %w", c.poolAddr, err)
	}

	c.Stop()
	c.PoolConn = poolConn
	c.poolAddr = addr

	go c.Run(context.Background())
	return nil
}

func (c *ProxyConn) proxyReader(ctx context.Context, sourceConn net.Conn, msgCh chan<- []byte) error {
	sourceReader := bufio.NewReader(sourceConn)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		msg, err := sourceReader.ReadBytes('\n')
		if err != nil {
			return lib.WrapError(ErrProxyRead, err)
		}

		// trim last newline char
		msg = msg[:len(msg)-1]

		if len(msg) <= 0 {
			continue
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		msgCh <- msg
	}
}

func (c *ProxyConn) proxyWriter(ctx context.Context, minerMsgCh <-chan []byte, poolMsgCh <-chan []byte) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-minerMsgCh:
			msg = c.transformer.ProcessMiningMessage(ctx, msg, c)
			if msg != nil {
				err := c.write(ctx, c.PoolConn, msg)
				if err != nil {
					return err
				}
			}
		case msg := <-poolMsgCh:
			msg = c.transformer.ProcessPoolMessage(ctx, msg, c)
			if msg != nil {
				err := c.write(ctx, c.minerConn, msg)
				if err != nil {
					return err
				}
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
	msg = append(msg, []byte("\n")...)
	_, err := c.minerConn.Write(msg)
	if err != nil {
		return lib.WrapError(ErrProxyWrite, err)
	}

	return nil
}

func (c *ProxyConn) WriteToPool(ctx context.Context, msg []byte) error {
	msg = append(msg, []byte("\n")...)
	_, err := c.PoolConn.Write(msg)
	if err != nil {
		return lib.WrapError(ErrProxyWrite, err)
	}

	return nil
}
