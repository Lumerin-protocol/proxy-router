package protocol

import (
	"context"
	"net"
	"sync"

	"go.uber.org/zap"
)

type StratumV1PoolPool struct {
	pool sync.Map
	conn *StratumV1Pool
	log  *zap.SugaredLogger
}

func NewStratumV1PoolPool(log *zap.SugaredLogger) *StratumV1PoolPool {
	return &StratumV1PoolPool{
		pool: sync.Map{},
		log:  log,
	}
}

func (p *StratumV1PoolPool) SetDest(addr string, authUser string, authPass string) error {
	conn, ok := p.load(addr)
	if ok {
		// TODO add lock
		p.conn = conn
		p.conn.ResendRelevantNotifications(context.TODO())
		p.log.Infof("conn reused %s", addr)

		return nil
	}

	c, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	p.log.Infof("Dialed dest %s", addr)

	conn = NewStratumV1Pool(c, p.log, authUser, authPass)

	err = conn.Connect()
	if err != nil {
		return err
	}

	// TODO add lock
	p.conn = conn
	p.store(addr, conn)
	return nil
}

func (p *StratumV1PoolPool) GetConn() *StratumV1Pool {
	return p.conn
}

func (p *StratumV1PoolPool) load(addr string) (*StratumV1Pool, bool) {
	conn, ok := p.pool.Load(addr)
	if !ok {
		return nil, false
	}
	return conn.(*StratumV1Pool), true
}

func (p *StratumV1PoolPool) store(addr string, conn *StratumV1Pool) {
	p.pool.Store(addr, conn)
}
