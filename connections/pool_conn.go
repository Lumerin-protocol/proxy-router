package connections

import (
	"net"
	"sync"

	"go.uber.org/zap"
)

type poolConn struct {
	handler MessageHandlerV2
	log     *zap.SugaredLogger
	connMap sync.Map
	conn    net.Conn
}

func (p *poolConn) DialDest(addr string) error {
	conn, ok := p.load(addr)
	if ok {
		// TODO add lock
		p.conn = conn
		return nil
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	// TODO add lock
	p.conn = conn
	p.store(conn)
	return nil
}

func (p *poolConn) Read(b []byte) (n int, err error) {
	return p.conn.Read(b)
}

func (p *poolConn) Write(b []byte) (n int, err error) {
	return p.conn.Write(b)
}

func (p *poolConn) load(addr string) (net.Conn, bool) {
	conn, err := p.connMap.Load(addr)
	return conn.(net.Conn), err
}

func (p *poolConn) store(conn net.Conn) {
	p.connMap.Store(conn.RemoteAddr(), conn)
}
