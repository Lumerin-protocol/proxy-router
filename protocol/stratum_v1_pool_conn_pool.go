package protocol

import (
	"context"
	"net"
	"sync"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

// Wraps the stratum miner pool connection to reuse multiple pool connections without handshake
type StratumV1PoolConnPool struct {
	pool sync.Map

	conn *StratumV1PoolConn
	mu   sync.Mutex // guards conn

	log interfaces.ILogger
}

func NewStratumV1PoolPool(log interfaces.ILogger) *StratumV1PoolConnPool {
	return &StratumV1PoolConnPool{
		pool: sync.Map{},
		log:  log,
	}
}

func (p *StratumV1PoolConnPool) GetDest() interfaces.IDestination {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.conn.GetDest()
}

func (p *StratumV1PoolConnPool) SetDest(dest interfaces.IDestination) error {
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

	conn, ok := p.load(dest.String())
	if ok {
		p.mu.Lock()
		p.conn = conn
		p.mu.Unlock()

		p.conn.ResendRelevantNotifications(context.TODO())
		p.log.Infof("conn reused %s", dest.String())

		return nil
	}
	p.log.Debugf("destination obj: %v; host: %v", dest, dest.GetHost())
	c, err := net.Dial("tcp", dest.GetHost())
	if err != nil {
		return err
	}
	p.log.Infof("Dialed dest %s", dest.GetHost())

	conn = NewStratumV1Pool(c, p.log, dest)

	err = conn.Connect()
	if err != nil {
		return err
	}

	p.mu.Lock()
	p.conn = conn
	p.mu.Unlock()

	p.store(dest.String(), conn)
	p.log.Infof("=========> dest was set %s", dest.GetHost())
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

var _ StratumV1DestConn = new(StratumV1PoolConnPool)
