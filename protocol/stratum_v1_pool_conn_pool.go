package protocol

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

// Wraps the stratum miner pool connection to reuse multiple pool connections without handshake
type StratumV1PoolConnPool struct {
	pool sync.Map

	conn        *StratumV1PoolConn
	mu          sync.Mutex // guards conn
	connTimeout time.Duration

	log        interfaces.ILogger
	logStratum bool
}

func NewStratumV1PoolPool(log interfaces.ILogger, connTimeout time.Duration, logStratum bool) *StratumV1PoolConnPool {
	return &StratumV1PoolConnPool{
		pool:        sync.Map{},
		connTimeout: connTimeout,
		log:         log,
		logStratum:  logStratum,
	}
}

func (p *StratumV1PoolConnPool) GetDest() interfaces.IDestination {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn == nil {
		return nil
	}
	return p.conn.GetDest()
}

func (p *StratumV1PoolConnPool) SetDest(ctx context.Context, dest interfaces.IDestination, configure *stratumv1_message.MiningConfigure) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	go func() {
		<-ctx.Done()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			p.log.Errorf("setDest context timeouted %s", ctx.Err())
		}
	}()

	p.mu.Lock()
	if p.conn != nil {
		if p.conn.GetDest().IsEqual(dest) {
			// noop if connection is the same
			p.log.Debug("dest wasn't changed, as it is the same")
			p.mu.Unlock()
			return nil
		}
	}

	p.mu.Unlock()

	// if p.conn != nil {
	// 	p.conn.PauseReading()
	// }

	// try to reuse connection from cache
	conn, ok := p.load(dest.String())
	if ok {

		p.setConn(conn)
		p.conn.ResendRelevantNotifications(context.TODO())
		p.log.Infof("conn reused %s", dest.String())

		return nil
	}

	// dial if not cached
	c, err := net.Dial("tcp", dest.GetHost())
	if err != nil {
		return err
	}
	p.log.Infof("dialed dest %s", dest)

	_ = c.(*net.TCPConn).SetLinger(60)

	conn = NewStratumV1Pool(c, p.log, dest, configure, p.connTimeout, p.logStratum)

	go func() {
		err := conn.Run(context.TODO())
		err2 := conn.Close()
		if err2 != nil {
			p.log.Errorf("pool connection closeout error, %s", err2)
		} else {
			p.log.Warnf("pool connection closed: %s", err)
		}
		p.pool.Delete(dest.String())
	}()

	ID := conn.GetDest().String()
	conn.Deadline(func() {
		// TODO: check if connection is not active before deleting
		// cause it may have not yet sent a submit but could be cleaned
		// causing miner to disconnect
		p.pool.Delete(ID)
		p.log.Debugf("connection was cleaned %s", ID)
	})

	err = conn.Connect()
	if err != nil {
		return err
	}

	conn.ResendRelevantNotifications(context.TODO())

	p.setConn(conn)

	p.store(dest.String(), conn)
	p.log.Infof("dest was set %s", dest)
	return nil
}

func (p *StratumV1PoolConnPool) Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error) {
	return p.conn.Read()
}

func (p *StratumV1PoolConnPool) Write(ctx context.Context, b stratumv1_message.MiningMessageGeneric) error {
	return p.conn.Write(ctx, b)
}

func (p *StratumV1PoolConnPool) GetExtranonce() (string, int) {
	return p.conn.GetExtranonce()
}

func (p *StratumV1PoolConnPool) load(addr string) (*StratumV1PoolConn, bool) {
	conn, ok := p.pool.Load(addr)
	if !ok {
		return nil, false
	}
	return conn.(*StratumV1PoolConn), true
}

func (p *StratumV1PoolConnPool) store(addr string, conn *StratumV1PoolConn) {
	p.pool.Store(addr, conn)
}

func (p *StratumV1PoolConnPool) getConn() *StratumV1PoolConn {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.conn
}

func (p *StratumV1PoolConnPool) setConn(conn *StratumV1PoolConn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.conn = conn
}

func (p *StratumV1PoolConnPool) ResendRelevantNotifications(ctx context.Context) {
	p.getConn().resendRelevantNotifications(ctx)
}

func (p *StratumV1PoolConnPool) SendPoolRequestWait(msg stratumv1_message.MiningMessageToPool) (*stratumv1_message.MiningResult, error) {
	return p.getConn().SendPoolRequestWait(msg)
}

func (p *StratumV1PoolConnPool) RegisterResultHandler(id int, handler StratumV1ResultHandler) {
	p.getConn().RegisterResultHandler(id, handler)
}

func (p *StratumV1PoolConnPool) RangeConn(f func(key any, value any) bool) {
	p.pool.Range(f)
}

func (p *StratumV1PoolConnPool) Close() error {
	p.pool.Range(func(key, value any) bool {
		poolConn := value.(*StratumV1PoolConn)
		err := poolConn.Close()
		if err != nil {
			p.log.Errorf("cannot close pool conn %s: %s", key, err)
		} else {
			p.log.Debugf("pool connection closed %s", key)
		}
		return true
	})

	p.pool = sync.Map{}

	return nil
}

var _ StratumV1DestConn = new(StratumV1PoolConnPool)
