package sockettcp

import (
	"context"
	"errors"
	"io"
	"net"
	"regexp"
	"strconv"
	"time"

	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

//
// Socket TCP
// Manages TCP socket connections acting as standard IO Read/Write package
//
//

const TCPAcceptChannelLen int = 2
const TCPReadChannelLen int = 10

const TCPReadBufferSize int = 4096

var ErrSocTCPListerClosed = errors.New("socket TCP: Listener closed")
var ErrSocTCPClosed = errors.New("socket TCP: socket closed")
var ErrSocTCPBadNetwork = errors.New("socket TCP: bad network protocol")
var ErrSocTCPEmtpyWriteBuf = errors.New("socket TCP: Emtpy Write Buffer")

//
// ----------------
//

type SocketStatusStruct struct {
	bytesRead    int
	bytesWritten int
	countRead    int
	countWrite   int
}

type ListenerStatusStruct struct {
	connectionCount int
}

//
// ----------------
//

type ListenTCPStruct struct {
	listener net.Listener
	ctx      context.Context
	cancel   func()
	accept   chan interface{}
	status   ListenerStatusStruct
}

type SocketTCPStruct struct {
	socket   net.Conn
	ctx      context.Context
	cancel   func()
	readchan chan readStruct
	status   SocketStatusStruct
}

type readStruct struct {
	err   error
	count int
	buf   []byte
}

//
// Err()
//
func (r *readStruct) Err() error {
	return r.err
}

//
// Buf()
//
func (r *readStruct) Buf() []byte {
	return r.buf
}

//
// Count()
//
func (r *readStruct) Count() int {
	return r.count
}

//
// Opens a listening socket on the port and IP address of the local system
//
// func NewListen(ctx context.Context, network string, addr string) (l interface{}, e error) {
func NewListen(ctx context.Context, network string, addr string) (l *ListenTCPStruct, e error) {

	contextlib.Logf(ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	//
	// network can be tcp, tcp4 or tcp6
	//
	switch network {
	case "tcp":
	case "tcp4":
	case "tcp6":
	default:
		return l, ErrSocTCPBadNetwork
	}

	ctx, cancel := context.WithCancel(ctx)
	c := cancel

	listenconfig := net.ListenConfig{}
	listener, e := listenconfig.Listen(ctx, network, addr)

	if e != nil {
		return l, e
	}

	// accept := make(chan *SocketTCPStruct, TCPAcceptChannelLen)
	accept := make(chan interface{}, TCPAcceptChannelLen)

	l = &ListenTCPStruct{
		listener: listener,
		ctx:      ctx,
		cancel:   c,
		accept:   accept,
		status: ListenerStatusStruct{
			connectionCount: 0,
		},
	}

	//
	// Go routine to run accept() on the socket and pass the connection back
	// via the channel.  The accept function can be canceled via the context
	//

	return l, e
}

//
//
//
func (l *ListenTCPStruct) Run() {
	//	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")
	go l.goListenAccept()
}

//
//
//
func (l *ListenTCPStruct) Ctx() context.Context {
	return l.ctx
}

//
// goAccept() go routine to accept connections and return new socket structs to the Accept() function
//
func (l *ListenTCPStruct) goListenAccept() {

	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	defer close(l.accept)

	for !l.Done() {
		// Network Accept()
		conn, e := l.listener.Accept()

		if e != nil {
			contextlib.Logf(l.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" listener.Accept() returned error:%s", e)
			break
		}

		if conn == nil {
			contextlib.Logf(l.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" Accept() returned empty connection")
			break
		}

		newsoc := createNewSocket(l.ctx, conn)

		l.accept <- newsoc
	}

	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" closing l.accept and exiting")
}

//
//
//
func (l *ListenTCPStruct) GetAcceptChan() <-chan interface{} {

	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	return l.accept

}

//
// Closes down a listening Socket
//
func (l *ListenTCPStruct) Close() {

	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	e := l.listener.Close()
	if e != nil {
		contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" listener.Close() returned an error:%s", e)
	}
}

//
//
//
func (l *ListenTCPStruct) Cancel() {

	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if l.cancel == nil {
		contextlib.Logf(l.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" cancel function is nil, struct:%v", l)
		return
	}

	l.cancel()
}

//
// Returns current status of the Listener
//
func (l *ListenTCPStruct) Addr() (addr net.Addr, e error) {

	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if l.Done() {
		return addr, ErrSocTCPListerClosed
	}

	addr = l.listener.Addr()

	return addr, e
}

//
// Returns address of the Listener
//
func (l *ListenTCPStruct) Status() (ltss *ListenerStatusStruct, e error) {

	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if l.Done() {
		return ltss, ErrSocTCPListerClosed
	}

	stat := l.status

	return &stat, e
}

//
//
//
func (s *ListenTCPStruct) Done() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

//
// Dial() creates a new TCP connection to the target address
// or returns an error
//
func Dial(ctx context.Context, network string, addr string) (s *SocketTCPStruct, e error) {

	contextlib.Logf(ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	//
	// network can be tcp, tcp4 or tcp6
	//
	switch network {
	case "tcp":
	case "tcp4":
	case "tcp6":
	default:
		return s, ErrSocTCPBadNetwork
	}

	//
	// CONTEXT Funkyness Here
	//
	cs := contextlib.GetContextStruct(ctx)
	// dialctx, cancel := context.WithCancel(ctx)
	// dialctx, cancel = context.WithTimeout(dialctx, time.Minute)
	dialctx, cancel := context.WithTimeout(ctx, time.Minute)
	dialctx = context.WithValue(dialctx, contextlib.ContextKey, cs)

	var conn net.Conn
	var d net.Dialer
	conn, e = d.DialContext(dialctx, network, addr)

	if e != nil {
		contextlib.Logf(ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" DialContext Error:%s", e)
		_ = cancel
		return nil, e
	} else {
		ctx = context.WithValue(ctx, contextlib.ContextKey, cs)
		s = createNewSocket(ctx, conn)
	}

	contextlib.Logf(ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Complete")

	return s, e
}

//
// Go Routine to listen to the context for cancel
//
func createNewSocket(ctx context.Context, conn net.Conn) (soc *SocketTCPStruct) {

	// contextlib.Logf(ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	rc := make(chan readStruct, TCPReadChannelLen)
	ctx, cancel := contextlib.CreateNewContext(ctx)
	soc = &SocketTCPStruct{
		socket:   conn,
		ctx:      ctx,
		cancel:   cancel,
		readchan: rc,
		// readbuf:  make([]byte, 0, TCPReadBufferSize),
		status: SocketStatusStruct{
			bytesRead:    0,
			bytesWritten: 0,
			countRead:    0,
			countWrite:   0,
		},
	}

	return soc
}

//
//
//
func (s *SocketTCPStruct) Cancel() {

	//	contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if s.Done() {
		contextlib.Logf(s.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" Already called")
		return
	}

	// close(s.readchan)
	s.cancel()
}

//
//
//
func (s *SocketTCPStruct) Ctx() context.Context {
	contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")
	return s.ctx
}

//
// Read()
// Blocks on getting a read back from readchan.
//
func (s *SocketTCPStruct) Read(buf []byte) (count int, e error) {

	// contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if s.Done() {
		return 0, ErrSocTCPClosed
	}

	count, e = s.socket.Read(buf)
	if e != nil {
		switch e {
		case io.ErrUnexpectedEOF:
			contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Socket Read() return Unexpected EOF")
		case io.EOF:
			contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Socket Read() return EOF")
		default:
			contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Socket Read() return default close")
			e = ErrSocTCPClosed
		}
		s.Close()
	}

	return count, e
}

//
//
//
func (s *SocketTCPStruct) Write(buf []byte) (count int, e error) {

	contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if s.Done() {
		contextlib.Logf(s.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" context closed")
		return 0, ErrSocTCPClosed
	}

	if len(buf) == 0 {
		contextlib.Logf(s.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" empty buffer")
		return 0, ErrSocTCPEmtpyWriteBuf
	}

	count, e = s.socket.Write(buf)

	if e == nil {
		s.status.bytesWritten += count
		s.status.countWrite++
	} else {
		contextlib.Logf(s.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" Write() error:%s", e)
	}

	return count, e
}

//
//
//
func (s *SocketTCPStruct) Status() (ss *SocketStatusStruct, e error) {

	contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if s.Done() {
		return nil, ErrSocTCPClosed
	}

	sss := s.status
	return &sss, e
}

//
//
//
func (s *SocketTCPStruct) Close() {

	//	contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if s.Done() {
		return
	}

	e := s.socket.Close()
	if e != nil {
		contextlib.Logf(s.ctx, contextlib.LevelDebug, lumerinlib.FileLineFunc()+" socket.Close() returned an error:%s", e)
	}

	s.Cancel()
}

//
// Returns the local address of the socket
//
func (s *SocketTCPStruct) LocalAddrString() (addr string, e error) {

	contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if s.Done() {
		return addr, ErrSocTCPClosed
	}

	return s.socket.LocalAddr().String(), e
}

//
// Returns the remote address of the socket
//
func (s *SocketTCPStruct) RemoteAddrString() (addr string, e error) {

	contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if s.Done() {
		return addr, ErrSocTCPClosed
	}

	return s.socket.RemoteAddr().String(), e
}

//
// Returns the local address of the socket
//
func (l *ListenTCPStruct) LocalAddr() (host string, port int, e error) {

	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	addr := l.listener.Addr().String()
	host, port, e = getAddr(l.ctx, addr)
	return
}

//
// Returns the local address of the socket
//
func (s *SocketTCPStruct) LocalAddr() (addr net.Addr, e error) {

	contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if s.Done() {
		return addr, ErrSocTCPClosed
	}

	addr = s.socket.LocalAddr()

	return addr, e
}

//
// Returns the local address of the socket
//
func (s *SocketTCPStruct) RemoteAddr() (addr net.Addr, e error) {

	//	contextlib.Logf(s.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if s.Done() {
		return addr, ErrSocTCPClosed
	}

	addr = s.socket.RemoteAddr()

	return addr, e
}

//
//
//
func (s *SocketTCPStruct) Done() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

//
// getAddr()
// Returns the local address of the socket
//
func getAddr(ctx context.Context, addr string) (host string, port int, e error) {

	contextlib.Logf(ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	regex := regexp.MustCompile("^(\\[*[a-fA-F0-9:]+\\]*):(\\d+)$")

	regexret := regex.FindStringSubmatch(addr)
	_ = regexret

	host = regexret[1]
	portstr := regexret[2]

	port, e = strconv.Atoi(portstr)

	return host, port, e
}
