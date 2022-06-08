package connectionmanager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/lumerinconnection"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

const DefaultDstSlots int = 8
const MaxDstSlots int = 16
const DefaultReadBufSize = 2048

// const DefaultReadEventChanSize = 10
const DefaultReadEventChanSize = 0

const SrcIdx int = -1
const DstIdx0 int = 0

var ErrConnMgrClosed = errors.New("CM: Closed")
var ErrConnDstClosed = errors.New("CM: Destination Closed")
var ErrConnMgrNoDefIndex = errors.New("CM: no default index")
var ErrConnMgrBadDefDest = errors.New("CM: bad default destination")
var ErrConnMgrBadDest = errors.New("CM: bad destination")
var ErrConnReadNotReady = errors.New("CM: there is nothing to read")
var ErrConnDstStillOpen = errors.New("CM: Dst Connection is still open")

//
// Listen Struct for new SRC connections coming in
//
type ConnectionListenStruct struct {
	ctx     context.Context
	cancel  func()
	lumerin *lumerinconnection.LumerinListenStruct
	port    int
	addr    net.Addr
	accept  chan *ConnectionStruct
}

//
// Struct for existing SRC connections and the associated outgoing DST connections
type ConnectionStruct struct {
	src      *lumerinconnection.LumerinSocketStruct
	dst      map[int]*lumerinconnection.LumerinSocketStruct
	defidx   int
	ctx      context.Context
	cancel   func()
	readChan chan *ConnectionReadEvent // Fed by all of the go routines servising the read events
}

//
// Send a single Read Event back up the stack with the
// SRC or DST index ID // -1 == SRC, 0+ == DST
type ConnectionReadEvent struct {
	index int
	data  []byte
	count int
	err   error
}

var dstCount chan int

//
// init()
// initializes the DstCounter
//
func init() {
	dstCount = make(chan int, 5)
	go goDstCounter(dstCount)
}

//
// goDstCounter()
// Generates a UniqueID for the destination handles
//
func goDstCounter(c chan int) {
	counter := 100
	for {
		c <- counter
		counter += 1
	}

}

func (c *ConnectionReadEvent) Index() int   { return c.index }
func (c *ConnectionReadEvent) Data() []byte { return c.data }
func (c *ConnectionReadEvent) Count() int   { return c.count }
func (c *ConnectionReadEvent) Err() error   { return c.err }

//
//
//
func NewListen(ctx context.Context) (cls *ConnectionListenStruct, e error) {

	contextlib.Logf(ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	ctx, cancel := context.WithCancel(ctx)
	_ = cancel

	addr := contextlib.GetSrc(ctx)

	l, e := lumerinconnection.NewListen(ctx, addr)
	if e != nil {
		contextlib.Logf(ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" NewListen() error:%s", e)
	} else {
		accept := make(chan *ConnectionStruct)
		cls = &ConnectionListenStruct{
			lumerin: l,
			ctx:     ctx,
			cancel:  cancel,
			port:    0,
			addr:    addr,
			accept:  accept,
		}
	}

	return cls, e
}

//
//
//
func (cls *ConnectionListenStruct) Run() {

	contextlib.Logf(cls.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	cls.lumerin.Run()
	go cls.goListenAccept()

}

//
//
//
func (cls *ConnectionListenStruct) getPort() (port int) {
	contextlib.Logf(cls.ctx, contextlib.LevelPanic, fmt.Sprint(lumerinlib.FileLineFunc()+" called"))
	return 0
}

//
//
//
func (cls *ConnectionListenStruct) getIp() net.Addr {
	return cls.addr
}

//
//
//
func (cls *ConnectionListenStruct) goListenAccept() {

	contextlib.Logf(cls.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	lumerinAcceptChan := cls.lumerin.GetAcceptChan()

FORLOOP:
	for {
		select {
		case <-cls.ctx.Done():
			contextlib.Logf(cls.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" context canceled")
			break FORLOOP
		case l := <-lumerinAcceptChan:
			ctx, cancel := context.WithCancel(cls.ctx)
			dst := map[int]*lumerinconnection.LumerinSocketStruct{}
			cs := &ConnectionStruct{
				src:      l,
				dst:      dst,
				defidx:   -1,
				ctx:      ctx,
				cancel:   cancel,
				readChan: make(chan *ConnectionReadEvent, DefaultReadEventChanSize),
			}

			cls.accept <- cs
			go cs.goRead(SrcIdx)
		}
	}

	contextlib.Logf(cls.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Exiting...")
}

//
//
//
func (cls *ConnectionListenStruct) Accept() <-chan *ConnectionStruct {

	contextlib.Logf(cls.ctx, contextlib.LevelTrace, fmt.Sprint(lumerinlib.FileLineFunc()+" called"))

	return cls.accept
}

//
//
//
func (cls *ConnectionListenStruct) Close() (e error) {
	contextlib.Logf(cls.ctx, contextlib.LevelTrace, fmt.Sprint(lumerinlib.FileLineFunc()+" called"))
	return cls.lumerin.Close()
}

//
//
//
func (cls *ConnectionListenStruct) Cancel() {
	contextlib.Logf(cls.ctx, contextlib.LevelTrace, fmt.Sprint(lumerinlib.FileLineFunc()+" closing down ConnectionListenStruct"))

	if cls.cancel == nil {
		contextlib.Logf(cls.ctx, contextlib.LevelError, fmt.Sprint(lumerinlib.FileLineFunc()+" cancel func is nil, struct:%v", cls))
		return
	}

	//close(cls.accept)
	cls.cancel()
}

//
// func (cs *ConnectionStruct) goRead()
// Reads from the lumerinconnection socket, packages it up and passes it to the readChan
//
func (cs *ConnectionStruct) goRead(index int) {

	//	contextlib.Logf(cs.ctx, contextlib.LevelTrace, fmt.Sprint(lumerinlib.FileLineFunc()+" enter - %d", index))

	var l *lumerinconnection.LumerinSocketStruct

	var name string
	if index < 0 {
		l = cs.src
		name = "SRC"
	} else {
		l = cs.dst[index]
		name = fmt.Sprintf("DST:%d", index)
	}

	if l == nil {
		contextlib.Logf(cs.ctx, contextlib.LevelError, fmt.Sprint(lumerinlib.FileLineFunc()+" %s bad index:%d", name, index))
		cre := &ConnectionReadEvent{
			index: index,
			data:  nil,
			count: 0,
			err:   ErrConnMgrBadDest,
		}

		// Getting panic here about a closed connection.
		// So there is data coming in, but the readChan has closed...
		if !cs.Done() {
			cs.readChan <- cre
			cs.Close()
		} else {
			contextlib.Logf(cs.ctx, contextlib.LevelError, fmt.Sprint(lumerinlib.FileLineFunc()+" we are in it deep now, cant event send an error up the stack"))
		}
		return
	}

FORLOOP:
	for !cs.Done() {

		data := make([]byte, DefaultReadBufSize)
		count, e := l.Read(data)
		data = data[:count]

		//
		// If there is an error from the Read, deal with it here
		//
		if e != nil {

			//
			// Notate the error Here
			//
			switch e {
			case io.EOF:
				contextlib.Logf(cs.ctx, contextlib.LevelInfo, fmt.Sprintf(lumerinlib.FileLineFunc()+" %s Read() index:%d returned EOF", name, index))
			case lumerinconnection.ErrLumConSocketClosed:
				contextlib.Logf(cs.ctx, contextlib.LevelInfo, fmt.Sprintf(lumerinlib.FileLineFunc()+" %s Read() index:%d returned %s", name, index, e))
			default:
				contextlib.Logf(cs.ctx, contextlib.LevelError, fmt.Sprintf(lumerinlib.FileLineFunc()+" %s Read() on index:%d returned error:%s", name, index, e))
			}

			// Src closed = shutdown the whole shebang and pass the error up the stack
			// if Dst closed pass the error up the stack
			// index = -1 (SRC) index >= 0 (DST)
			if index < 0 {
				e = ErrConnMgrClosed
			} else {
				e = ErrConnDstClosed
			}
		}

		cre := &ConnectionReadEvent{
			index: index,
			data:  data,
			count: count,
			err:   e,
		}

		// Getting panic here about a closed connection.
		// So there is data coming in, but the readChan has closed...
		if !cs.Done() {
			cs.readChan <- cre
		}

		if e != nil {
			contextlib.Logf(cs.ctx, contextlib.LevelError, fmt.Sprintf(lumerinlib.FileLineFunc()+" %s UID:%d Error:%s", name, e))
			break FORLOOP
		}
	}

	// Something errored or closed, so call close to be sure nothing is hanging
	contextlib.Logf(cs.ctx, contextlib.LevelTrace, fmt.Sprintf(lumerinlib.FileLineFunc()+" %s UID:%d Exiting", name, index))

	if index < 0 {
		cs.Close()
	}

}

//
//
//
func (cs *ConnectionStruct) GetReadChan() <-chan *ConnectionReadEvent {
	if cs == nil {
		panic(lumerinlib.FileLineFunc() + " ConnectionStruct is nil")
	}

	// contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	return cs.readChan
}

//
// Close() will close out all src and dst connections via the cancel context function
//
func (cs *ConnectionStruct) Close() {

	// 	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	// Close out all of the Lumerin connections
	cs.src.Close()
	for i := 0; i < len(cs.dst); i++ {
		cs.dst[i].Close()
	}

	cs.Cancel() // This should close all open src and dst connections

}

//
//
//
func (cs *ConnectionStruct) Cancel() {

	//	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if cs.Done() {
		//		contextlib.Logf(cs.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" called already")
		return
	}

	if cs.cancel == nil {
		//		contextlib.Logf(cs.ctx, contextlib.LevelError, fmt.Sprintf(lumerinlib.FileLineFunc()+" cancel func it nil, struct:%v", cs))
		return
	}

	//close(cs.readChan)
	cs.cancel()
}

//
// Dial() opens up a new dst connection and inserts it into the first avalable dst slot
// If this is the 0th slow, the default dst is set as well
//
// func (cs *ConnectionStruct) Dial(ctx context.Context, port int, ip net.IPAddr) (idx int, e error) {
func (cs *ConnectionStruct) Dial(addr net.Addr) (idx int, e error) {

	if cs == nil {
		panic("ConnectionStruct is nil...")
	}

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, fmt.Sprint(lumerinlib.FileLineFunc()+" called"))

	idx = <-dstCount

	dst, e := lumerinconnection.Dial(cs.ctx, addr)
	if e != nil {
		contextlib.Logf(cs.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" IDX: %d - Dial error:%s", idx, e)
		return idx, e
	}

	// Verify the slot is empty
	if cs.dst[idx] != nil {
		contextlib.Logf(cs.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" cannot be here, idx:%d", idx)
	}

	cs.dst[idx] = dst
	go cs.goRead(idx)

	return idx, nil
}

//
//
//
func (cs *ConnectionStruct) GetRemoteAddrIdx(idx int) (addr net.Addr, e error) {

	if cs == nil {
		panic("ConnectionStruct is nil...")
	}

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, fmt.Sprintf(lumerinlib.FileLineFunc()+" called idx:%d", idx))

	if idx < 0 {
		addr, e = cs.src.GetRemoteAddr()
	} else {
		_, ok := cs.dst[idx]
		if !ok {
			contextlib.Logf(cs.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" Bad IDX:%d, dst:%v", idx, cs.dst)
			return nil, ErrConnMgrBadDest
		}

		// Verify the slot is NOT empty
		if cs.dst[idx] == nil {
			contextlib.Logf(cs.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" Bad IDX:%d", idx)
			return nil, ErrConnMgrBadDest
		}

		addr, e = cs.dst[idx].GetRemoteAddr()
	}

	return addr, e

}

//
// ReDialIdx() will attempt to reconnect to the same dst, first checking the the line is closed
// It is used in case a connection is severed
//
func (cs *ConnectionStruct) ReDialIdx(idx int) (e error) {

	if cs == nil {
		panic("ConnectionStruct is nil...")
	}

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, fmt.Sprintf(lumerinlib.FileLineFunc()+" called idx:%d", idx))

	if idx < 0 {
		contextlib.Logf(cs.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" Bad IDX:%d", idx)
		return ErrConnMgrBadDest
	}

	_, ok := cs.dst[idx]
	if !ok {
		contextlib.Logf(cs.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" Bad IDX:%d, dst:%v", idx, cs.dst)
		return ErrConnMgrBadDest
	}

	// Verify the slot is NOT empty
	if cs.dst[idx] == nil {
		contextlib.Logf(cs.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" cannot be here, idx:%d", idx)
	}

	addr, e := cs.dst[idx].GetRemoteAddr()
	if e != nil {
		contextlib.Logf(cs.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" GetRemoteAddr() IDX:%d, error:%s", idx, e)
		return ErrConnMgrBadDest
	}

	if !cs.dst[idx].Done() {
		cs.dst[idx].Close()
	}

	dst, e := lumerinconnection.Dial(cs.ctx, addr)
	if e != nil {
		contextlib.Logf(cs.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" IDX: %d -(re)Dial error:%s", idx, e)
		return e
	}

	cs.dst[idx] = dst

	return nil
}

//
//
//
func (cs *ConnectionStruct) SetRoute(idx int) (e error) {

	// contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if idx < 0 || cs.dst[idx] == nil {
		e = ErrConnMgrBadDest
		return e
	}

	cs.defidx = idx

	return nil

}

//
//
//
func (cs *ConnectionStruct) GetRoute() (idx int, e error) {

	// contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	idx = cs.defidx

	if cs.defidx < 0 {
		e = ErrConnMgrBadDest
	}

	return idx, e

}

//
//
//
func (cs *ConnectionStruct) SrcGetSocket() (s *lumerinconnection.LumerinSocketStruct, e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	return cs.src, nil
}

//
//
//
func (cs *ConnectionStruct) SrcGetRemoteAddr() (addr net.Addr, e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	return cs.src.GetRemoteAddr()
}

//
//
//
func (cs *ConnectionStruct) SrcRead(buf []byte) (count int, e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	return cs.src.Read(buf)
}

//
//
//
func (cs *ConnectionStruct) SrcWrite(buf []byte) (count int, e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	return cs.src.Write(buf)

}

//
// SrcClose() calls (*CS) Close() to close everything down
//
func (cs *ConnectionStruct) SrcClose() {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	cs.Close()
}

//
// Get the first connected socket struct back
//
func (cs *ConnectionStruct) DstGetSocket() (s *lumerinconnection.LumerinSocketStruct, e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if cs.defidx < 0 {
		e = ErrConnMgrNoDefIndex
		return nil, e
	}

	if cs.dst[cs.defidx] == nil {
		e = ErrConnMgrBadDefDest
		return nil, e
	}

	return cs.dst[cs.defidx], e
}

//
//
//
func (cs *ConnectionStruct) DstGetRemoteAddr() (addr net.Addr, e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if cs.defidx < 0 {
		e = ErrConnMgrNoDefIndex
		return nil, e
	}

	if cs.dst[cs.defidx] == nil {
		e = ErrConnMgrBadDefDest
		return nil, e
	}

	return cs.dst[cs.defidx].GetRemoteAddr()
}

//
//
//
func (cs *ConnectionStruct) DstRead(buf []byte) (count int, e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if cs.defidx < 0 {
		e = ErrConnMgrNoDefIndex
		return 0, e
	}

	if cs.dst[cs.defidx] == nil {
		e = ErrConnMgrBadDefDest
		return 0, e

	}

	return cs.dst[cs.defidx].Read(buf)

}

//
//
//
func (cs *ConnectionStruct) DstWrite(buf []byte) (count int, e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if cs.defidx < 0 {
		e = ErrConnMgrNoDefIndex
		return 0, e
	}

	if cs.dst[cs.defidx] == nil {
		e = ErrConnMgrBadDefDest
		return 0, e

	}

	return cs.dst[cs.defidx].Write(buf)

}

//
//
//
func (cs *ConnectionStruct) DstClose() (e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if cs.defidx < 0 {
		e = ErrConnMgrNoDefIndex
		return e
	}

	if cs.dst[cs.defidx] == nil {
		e = ErrConnMgrBadDefDest
		return e

	}

	return cs.dst[cs.defidx].Close()

}

//
//
//
func (cs *ConnectionStruct) IdxGetSocket(idx int) (s *lumerinconnection.LumerinSocketStruct, e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if cs.defidx < 0 {
		e = ErrConnMgrNoDefIndex
		return nil, e
	}

	if cs.dst[idx] == nil {
		e = ErrConnMgrBadDest
		return nil, e
	}

	return cs.dst[idx], e
}

//
//
//
func (cs *ConnectionStruct) IdxGetRemoteAddr(idx int) (addr net.Addr, e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if cs.defidx < 0 {
		e = ErrConnMgrNoDefIndex
		return nil, e
	}

	if cs.dst[cs.defidx] == nil {
		e = ErrConnMgrBadDefDest
		return nil, e
	}

	return cs.dst[idx].GetRemoteAddr()
}

//
//
//
func (cs *ConnectionStruct) IdxRead(idx int, buf []byte) (count int, e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if idx < 0 {
		e = ErrConnMgrNoDefIndex
		return 0, e
	}

	if cs.dst[idx] == nil {
		e = ErrConnMgrBadDest
		return 0, e
	}

	return cs.dst[idx].Read(buf)

}

//
//
//
func (cs *ConnectionStruct) IdxWrite(idx int, buf []byte) (count int, e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, fmt.Sprintf(lumerinlib.FileLineFunc()+" called on idx: %d", idx))

	if idx < 0 {
		e = ErrConnMgrNoDefIndex
		return 0, e
	}

	if cs.dst[idx] == nil {
		e = ErrConnMgrBadDefDest
		return 0, e

	}

	return cs.dst[idx].Write(buf)

}

//
//
//
func (cs *ConnectionStruct) IdxClose(idx int) (e error) {

	contextlib.Logf(cs.ctx, contextlib.LevelTrace, fmt.Sprintf(lumerinlib.FileLineFunc()+" called"))

	if idx < 0 {
		e = ErrConnMgrNoDefIndex
		return e
	}

	if cs.dst[idx] == nil {
		e = ErrConnMgrBadDefDest
		return e

	}

	if idx == cs.defidx {
		cs.defidx = -1
	}

	e = cs.dst[idx].Close()
	cs.dst[idx] = nil

	return e
}

//
//
//
func (cs *ConnectionStruct) Done() bool {
	select {
	case <-cs.ctx.Done():
		return true
	default:
		return false
	}
}
