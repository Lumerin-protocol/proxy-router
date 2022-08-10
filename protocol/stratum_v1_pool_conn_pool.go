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

func (p *StratumV1PoolConnPool) SetDest(addr string, authUser string, authPass string) error {
	p.mu.Lock()
	if p.conn != nil {
		if a, u, pwd := p.conn.GetDest(); a == addr && u == authUser && pwd == authPass {
			// noop if connection is the same
			p.log.Debug("dest wasn't changed, as it is the same")
			p.mu.Unlock()
			return nil
		}
	}
	p.mu.Unlock()

	// TODO: maintain separate connections per address+user
	conn, ok := p.load(addr)
	if ok {
		// TODO add lock
		p.mu.Lock()
		p.conn = conn
		p.mu.Unlock()

		p.conn.ResendRelevantNotifications(context.TODO())
		p.log.Infof("conn reused %s", addr)

		return nil
	}

	c, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	p.log.Infof("Dialed dest %s", addr)

	conn = NewStratumV1Pool(c, p.log, addr, authUser, authPass)

	err = conn.Connect()
	if err != nil {
		return err
	}

	p.mu.Lock()
	p.conn = conn
	p.mu.Unlock()

	p.store(addr, conn)
	p.log.Infof("=========> dest was set %s", addr)
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
