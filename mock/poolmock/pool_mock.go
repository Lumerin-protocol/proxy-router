package poolmock

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

const (
	DEFAULT_DIFF = 50000
)

type PoolMock struct {
	listener net.Listener
	log      interfaces.ILogger

	port        int
	defaultDiff int

	connMap  sync.Map
	workerHr sync.Map
}

// NewPoolMock creates new mock pool. Set port to zero to auto-select available port. Watch also GetPort()
func NewPoolMock(port int, log interfaces.ILogger) *PoolMock {
	return &PoolMock{
		log:         log,
		port:        port,
		defaultDiff: DEFAULT_DIFF,
	}
}

func (p *PoolMock) SetDefaultDiff(diff int) {
	p.defaultDiff = diff
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
	errCh := make(chan error)

	go func() {
		errCh <- p.run(ctx)
	}()

	select {
	case <-ctx.Done():
		p.close()
		return ctx.Err()
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			p.close()
		}
		return err
	}
}

func (p *PoolMock) run(ctx context.Context) error {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			return err
		}

		go func() {
			connID := conn.RemoteAddr().String()
			p.log.Infof("new miner connection %s", connID)

			poolMockConn := NewPoolMockConn(conn, p.defaultDiff, p.OnSubmit, p.log.Named(connID))
			p.storeConn(poolMockConn)

			err = poolMockConn.Run(ctx)
			if err != nil {
				p.log.Warnf("connection error: %s", err)
			}

			p.removeConn(poolMockConn.ID())
			_ = conn.Close()
			p.log.Infof("miner connection closed %s", connID)
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

func (p *PoolMock) GetHRByWorkerName(workerName string) (hrGHS int, ok bool) {
	hr, ok := p.workerHr.Load(workerName)
	if !ok {
		return 0, false
	}
	return hr.(*hashrate.Hashrate).GetHashrate5minAvgGHS(), true
}

func (p *PoolMock) OnSubmit(workerName string, diff int64) {
	hr, _ := p.workerHr.LoadOrStore(workerName, hashrate.NewHashrate(p.log))
	workerHr := hr.(*hashrate.Hashrate)
	workerHr.OnSubmit(diff)
	p.log.Infof("got submit from worker(%s), hrGHS %d", workerName, workerHr.GetHashrate5minAvgGHS())
}

func (p *PoolMock) close() {
	p.connMap.Range(func(key, value any) bool {
		conn := value.(*PoolMockConn)
		_ = conn.conn.Close()
		return true
	})

	_ = p.listener.Close()
}
