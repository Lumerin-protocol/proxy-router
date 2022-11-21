package poolmock

import (
	"context"
	"fmt"
	"net"
	"sync"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
)

type PoolMock struct {
	listener net.Listener
	port     int
	connMap  sync.Map

	log interfaces.ILogger
}

// NewPoolMock creates new mock pool. Set port to zero to auto-select available port. Watch also GetPort()
func NewPoolMock(port int) *PoolMock {
	return &PoolMock{
		log:  lib.NewTestLogger(),
		port: port,
	}
}

func (p *PoolMock) Connect(ctx context.Context) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", p.port))
	if err != nil {
		return err
	}

	p.listener = listener
	p.port = parsePort(listener.Addr())

	p.log.Infof("pool mock is listening on port %d", p.port)
	return nil
}

func (p *PoolMock) Run(ctx context.Context) error {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			return err
		}

		go func() {
			p.log.Infof("new miner connection %s", conn.LocalAddr().String())

			poolMockConn := NewPoolMockConn(conn, p.log.Named(conn.LocalAddr().String()))
			p.storeConn(poolMockConn)

			err = poolMockConn.Run(ctx)
			if err != nil {
				p.log.Warnf("connection error: %s", err)
			}

			p.removeConn(poolMockConn.ID())
			_ = conn.Close()
			p.log.Infof("miner connection closed %s", conn.LocalAddr().String())
		}()
	}
}

// GetPort returns port which server uses
func (p *PoolMock) GetPort() int {
	return p.port
}

func (p *PoolMock) storeConn(conn *PoolMockConn) {
	p.connMap.Store(conn.ID(), conn)
}

func (p *PoolMock) removeConn(ID string) {
	p.connMap.Delete(ID)
}

func (p *PoolMock) GetConnByWorkerName(workerName string) *PoolMockConn {
	var foundConn *PoolMockConn

	p.connMap.Range(func(key, value any) bool {
		conn := value.(*PoolMockConn)
		if conn.GetWorkerName() == workerName {
			foundConn = conn
			return false
		}
		return true
	})

	return foundConn
}
