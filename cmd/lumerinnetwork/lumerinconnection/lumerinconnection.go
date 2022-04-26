package lumerinconnection

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/sockettcp"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

//
// Lumerinconnection is a package that can interpret connection requests to see if
// there is already a Lumerin Protocol Trunk in place with the target node
// and redirect the connection across the trunk instead of a new direct connection.
//
// Most of the functionality will be implemented in the future, at this point it
// only supports direct TCP connections.  It is easier to have this here now, then
// to reto-fit it in later.
//
//

const LumerinAcceptChannelLen int = 2
const LumerinReadChannelLen int = 10

var ErrLumConListenClosed = errors.New("Lumerin Connection Listen Socket closed")
var ErrLumConSocketClosed = errors.New("Lumerin Connection Listen Socket closed")

type LumProto string

const TCP LumProto = "tcp"
const TCP4 LumProto = "tcp4"
const TCP6 LumProto = "tcp6"
const UDP LumProto = "udp"
const UDP4 LumProto = "udp4"
const UDP6 LumProto = "udp6"
const TRUNK LumProto = "trunk"
const TCPTRUNK LumProto = "tcptrunk"
const UDPTRUNK LumProto = "udptrunk"
const ANYAVAILABLE LumProto = "anyavailable"

type LumerinListenerInterface interface {
	GetAcceptChan() <-chan interface{}
	Close()
	// Ctx() context.Context
	// Cancel()
	// Addr() (net.Addr, error)
	// Status()
	// LocalAddr() (string, int, error)
}

//
// This will contain a regular socket or virtual socket structure
//
type LumerinListenStruct struct {
	ctx      context.Context
	cancel   func()
	listener LumerinListenerInterface
	accept   chan *LumerinSocketStruct
}

type LumerinSocketStruct struct {
	ctx        context.Context
	cancel     func()
	remoteaddr net.Addr
	socket     interface{}
}

//
// Do this here or at a lower layer?
//
type LumerinListenStatusStruct struct {
	connectionCount int
}

type LumerinConnectionStatusStruct struct {
	bytesRead    int
	bytesWritten int
	countRead    int
	countWrite   int
}

var ErrLumSocClosed = errors.New("lumerin socket: virt socket closed")

//
// Setup listening on the port/IP and on the Lumerin Port
// -- setsup the listening routine with cancel context
//
// Needs to be replaced with net.Addr
//
// func Listen(ctx context.Context, p LumProto, port int, ip net.IPAddr) (l *LumerinListenStruct, e error) {
func NewListen(ctx context.Context, addr net.Addr) (l *LumerinListenStruct, e error) {

	contextlib.Logf(ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	ctx, cancel := context.WithCancel(ctx)

	proto := addr.Network()
	ipaddr := addr.String()

	// Lots of error checking here

	lumproto := LumProto(proto)

	switch lumproto {
	case TCP:
		fallthrough
	case TCP4:
		fallthrough
	case TCP6:
		var tcp *sockettcp.ListenTCPStruct
		tcp, e = sockettcp.NewListen(ctx, proto, ipaddr)
		if e != nil {
			contextlib.Logf(ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" NewListen() error:%s", e)
		} else {
			accept := make(chan *LumerinSocketStruct)
			l = &LumerinListenStruct{
				ctx:      ctx,
				cancel:   cancel,
				listener: tcp,
				accept:   accept,
			}
		}

	case UDP:
		fallthrough
	case UDP4:
		fallthrough
	case UDP6:
		fallthrough
	case TRUNK:
		fallthrough
	case TCPTRUNK:
		fallthrough
	case UDPTRUNK:
		fallthrough
	case ANYAVAILABLE:
		contextlib.Logf(ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Protocol not implemented, yet:%s", string(lumproto))

	default:
		contextlib.Logf(ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Proto:'%s' not supported\n", lumproto)
	}

	return l, e
}

//
//
//
func (ll *LumerinListenStruct) Run() {

	if ll.ctx == nil {
		panic(lumerinlib.FileLineFunc() + " ctx == nil")
	}

	contextlib.Logf(ll.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	switch ll.listener.(type) {
	case *sockettcp.ListenTCPStruct:
		ll.listener.(*sockettcp.ListenTCPStruct).Run()

	default:
		contextlib.Logf(ll.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Default reached, type: %T", ll.listener)
	}

	go ll.goListenAccept()

}

//
// reads the acceptChan for new connections, or the channel closure
//
func (ll *LumerinListenStruct) goListenAccept() {

	contextlib.Logf(ll.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	acceptChan := ll.listener.GetAcceptChan()
FORLOOP:
	for {
		select {
		case <-ll.ctx.Done():
			break FORLOOP
		case socket := <-acceptChan:
			if socket == nil {
				contextlib.Logf(ll.ctx, contextlib.LevelWarn, lumerinlib.FileLineFunc()+" Accept() returned nil, assumed closed")
				break FORLOOP
			}

			var addr net.Addr
			var e error

			switch socket.(type) {
			case *sockettcp.SocketTCPStruct:
				addr, e = socket.(*sockettcp.SocketTCPStruct).RemoteAddr()
				if e != nil {
					contextlib.Logf(ll.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" RemoteAddr() error:%s", e)
				}

			default:
				contextlib.Logf(ll.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" socket type: %t not supported", socket)
			}

			lci := &LumerinSocketStruct{
				ctx:        ll.ctx,
				socket:     socket,
				remoteaddr: addr,
			}
			ll.accept <- lci

		}

	}

	contextlib.Logf(ll.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Exiting...")
}

//
//
//
func (ll *LumerinListenStruct) GetAcceptChan() <-chan *LumerinSocketStruct {

	contextlib.Logf(ll.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	return ll.accept

}

//
//
//
func (ll *LumerinListenStruct) Close() (e error) {

	contextlib.Logf(ll.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	switch ll.listener.(type) {
	case *sockettcp.ListenTCPStruct:
		ll.listener.(*sockettcp.ListenTCPStruct).Close()
	default:
		contextlib.Logf(ll.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Default reached, type: %T", ll.listener)
	}

	ll.cancel()
	return e
}

//
//
//
func (ll *LumerinListenStruct) Cancel() {
	contextlib.Logf(ll.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")
	if ll.Done() {
		contextlib.Logf(ll.ctx, contextlib.LevelInfo, lumerinlib.FileLineFunc()+" already called")
		return
	}

	if ll.cancel == nil {
		contextlib.Logf(ll.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" cancel function is nil, struct:%v", ll)
	}

	//close(ll.accept)
	ll.cancel()
}

//
//
//
func (ll *LumerinListenStruct) Done() bool {
	select {
	case <-ll.ctx.Done():
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------------------------
//

//
//
//
func Dial(ctx context.Context, addr net.Addr) (lci *LumerinSocketStruct, e error) {

	contextlib.Logf(ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	proto := addr.Network()
	ipaddr := addr.String()

	var split []string
	split = strings.Split(ipaddr, ":")
	if len(split) < 2 {
		contextlib.Logf(ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" Split() returned len:%d:", len(split))
		return nil, fmt.Errorf(lumerinlib.FileLineFunc()+" Split returned len %d", len(split))
	}
	host := split[0]
	port := split[1]
	if port == "0" {
		contextlib.Logf(ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" port == 0")
		return nil, fmt.Errorf(lumerinlib.FileLineFunc() + " port == 0")
	}
	if host == "" {
		contextlib.Logf(ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" host == ''")
		return nil, fmt.Errorf(lumerinlib.FileLineFunc() + " host == ''")
	}

	// Lots of error checking here

	lumproto := LumProto(proto)

	switch lumproto {
	case TCP:
		fallthrough
	case TCP4:
		fallthrough
	case TCP6:
		var tcp *sockettcp.SocketTCPStruct
		tcp, e = sockettcp.Dial(ctx, string(lumproto), ipaddr)
		if e != nil {
			contextlib.Logf(ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Dial() error returned: %v", e)
			return nil, e
		}
		ctx, cancel := context.WithCancel(ctx)
		lci = &LumerinSocketStruct{
			ctx:        ctx,
			cancel:     cancel,
			socket:     tcp,
			remoteaddr: addr,
		}

	case UDP:
		fallthrough
	case UDP4:
		fallthrough
	case UDP6:
		fallthrough
	case TRUNK:
		fallthrough
	case TCPTRUNK:
		fallthrough
	case UDPTRUNK:
		fallthrough
	case ANYAVAILABLE:
		contextlib.Logf(ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Protocol not implemented yet: %s", lumproto)

	default:
		contextlib.Logf(ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Default reached, lumproto: %s", lumproto)
	}

	return lci, e
}

//
//
//
func (l *LumerinSocketStruct) Read(buf []byte) (count int, e error) {

	// contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if l.Done() {
		return 0, ErrLumConSocketClosed
	}

	switch l.socket.(type) {
	case *sockettcp.SocketTCPStruct:
		count, e = l.socket.(*sockettcp.SocketTCPStruct).Read(buf)
		if e != nil {
			switch e {
			case io.EOF:
			case io.ErrUnexpectedEOF:
			case sockettcp.ErrSocTCPClosed:
			default:
				contextlib.Logf(l.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" Read() returned unexpected error: %s", e)
			}

			contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Read() returned unexpected error: %s", e)

			l.Close()
			return 0, ErrLumConSocketClosed
		}
	default:
		contextlib.Logf(l.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Default reached, type: %T", l.socket)
	}

	return count, e
}

//
// Write()
//
func (l *LumerinSocketStruct) Write(buf []byte) (count int, e error) {

	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if l.Done() {
		return 0, ErrLumConSocketClosed
	}

	switch l.socket.(type) {
	case *sockettcp.SocketTCPStruct:
		count, e = l.socket.(*sockettcp.SocketTCPStruct).Write(buf)
		if e != nil {
			switch e {
			case io.EOF:
			case io.ErrUnexpectedEOF:
			case sockettcp.ErrSocTCPClosed:
			case sockettcp.ErrSocTCPEmtpyWriteBuf:
			default:
				contextlib.Logf(l.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" Write() returned unexpected error: %s", e)
			}
			l.Close()
			return 0, ErrLumConSocketClosed
		}
	default:
		contextlib.Logf(l.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Default reached, type: %T", l.socket)
	}

	return count, e
}

//
//
//
func (l *LumerinSocketStruct) Status() (stat *LumerinConnectionStatusStruct, e error) {

	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if l.Done() {
		return nil, ErrLumConSocketClosed
	}

	switch l.socket.(type) {
	case *sockettcp.SocketTCPStruct:
		_, e = l.socket.(*sockettcp.SocketTCPStruct).Status()

		if e != nil {
			contextlib.Logf(l.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" Status() returned error: %s", e)
		}
		// Fill in here
		stat = &LumerinConnectionStatusStruct{}

	default:
		contextlib.Logf(l.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Default reached, type: %T", l.socket)
	}

	return stat, e
}

//
//
//
func (l *LumerinSocketStruct) Close() (e error) {

	if l == nil {
		return errors.New(lumerinlib.FileLineFunc() + " nil pointer ")
	}

	//	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if l.Done() {
		return ErrLumConSocketClosed
	}

	switch l.socket.(type) {
	case *sockettcp.SocketTCPStruct:
		l.socket.(*sockettcp.SocketTCPStruct).Close()
	default:
		contextlib.Logf(l.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Default reached, type: %T", l.socket)
	}

	l.Cancel()

	return e
}

//
//
//
func (l *LumerinSocketStruct) GetAddr() net.Addr {
	return l.remoteaddr
}

//
//
//
func (l *LumerinSocketStruct) Cancel() {

	//	contextlib.Logf(l.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if l.Done() {
		contextlib.Logf(l.ctx, contextlib.LevelInfo, lumerinlib.FileLineFunc()+" already called")
		return
	}

	if l.cancel == nil {
		//		contextlib.Logf(l.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" cancel function is nul, struct:%v", l)
		return
	}

	l.cancel()
}

//
// Done()
// return socket closed error if the context shows done
//
func (l *LumerinSocketStruct) Done() bool {
	select {
	case <-l.ctx.Done():
		return true
	default:
		return false
	}
}
