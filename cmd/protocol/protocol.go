package protocol

import (
	"context"
	"errors"
	"fmt"
	"net"

	simple "gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/SIMPL"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

//
// Top layer protocol template functions that a new protocol will use to access the SIMPLe layer
//

var ErrDefaultRouteNotSet = errors.New("Default Route Not Set")

type ProtocolListenStruct struct {
	ctx          context.Context
	cancel       func()
	simplelisten *simple.SimpleListenStruct
	accept       chan *ProtocolStruct
}

type ProtocolStruct struct {
	ctx       context.Context
	cancel    func()
	simple    *simple.SimpleStruct
	eventchan chan *simple.SimpleEvent
	srcconn   *ProtocolConnectionStruct
	dstconn   *ProtocolDstStruct
}

//
// NewListen() Create a new ProtocolListenStruct
// Opens the default destination
//
func NewListen(ctx context.Context) (pls *ProtocolListenStruct, e error) {

	contextlib.Logf(ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	//
	// Basic error checking, make sure that the ContextStruct is
	// filled out correctly
	//
	cs := contextlib.GetContextStruct(ctx)
	if cs == nil {
		contextlib.Logf(ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Context Structre not present")
	}

	if cs.GetMsgBus() == nil {
		cs.Logf(contextlib.LevelPanic, "Context MsgBus not defined")
	}
	if cs.GetSrc() == nil {
		cs.Logf(contextlib.LevelPanic, "Context Src Addr not defined")
	}
	if cs.GetDest() == nil {
		cs.Logf(contextlib.LevelPanic, "Context DstID not defined")
	}

	ctx, cancel := context.WithCancel(ctx)

	sls, e := simple.NewListen(ctx)

	if e != nil {
		contextlib.Logf(ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" Lumerin Listen() return error:%s", e)
	} else {
		accept := make(chan *ProtocolStruct)
		pls = &ProtocolListenStruct{
			ctx:          ctx,
			cancel:       cancel,
			simplelisten: sls,
			accept:       accept,
		}
	}

	return pls, e
}

//
//
//
func (p *ProtocolListenStruct) GetAccept() <-chan *ProtocolStruct {
	return p.accept
}

//
// Ctx() returns the current context of the ProtocolListenStruct
//
func (p *ProtocolListenStruct) Ctx() context.Context {
	return p.ctx
}

//
// Cancel() calls the simple layer Cancel function on the SimpleListenStruct
//
func (pls *ProtocolListenStruct) Cancel() {

	contextlib.Logf(pls.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if pls.cancel == nil {
		contextlib.Logf(pls.ctx, contextlib.LevelError, fmt.Sprint(lumerinlib.FileLineFunc()+" cancel func is nil, struct:%v", pls))
		return
	}

	pls.cancel()
}

//
// Run() calls the simple layer Run function on the SimpleListenStruct
//
func (pls *ProtocolListenStruct) Run() {

	//	contextlib.Logf(pls.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	pls.simplelisten.Run()
	go pls.goListenAccept()
}

//
// goAccept()
// Listens for the Accept event from the SIMPL layer
// The simpleStruct has already been inialized
//
func (pls *ProtocolListenStruct) goListenAccept() {

	contextlib.Logf(pls.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	newSimpleStructChan := pls.simplelisten.GetAccept()

FORLOOP:
	for {
		select {
		case <-pls.ctx.Done():
			break FORLOOP
		case newSimpleStruct := <-newSimpleStructChan:
			// contextlib.Logf(pls.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" simplelisten.Accept() recieved")

			if newSimpleStruct == nil {
				contextlib.Logf(pls.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" pls.simplelisten.Accept() stopping")
				pls.Cancel()
				break FORLOOP
			}
			if nil == newSimpleStruct.Ctx() {
				contextlib.Logf(pls.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" newSimpleStruct empty CTX stopping")
				pls.Cancel()
				break FORLOOP
			}
			if nil == newSimpleStruct.ConnectionStruct {
				contextlib.Logf(pls.ctx, contextlib.LevelError, lumerinlib.FileLineFunc()+" newSimpleStruct empty ConnectionStruct stopping")
				pls.Cancel()
				break FORLOOP
			}

			ps, e := NewProtocol(pls.ctx, newSimpleStruct)
			if e != nil {
				contextlib.Logf(pls.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" NewProtocol() error:%s", e)
			}
			if ps == nil {
				contextlib.Logf(pls.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" NewProtocol() returned nil, assuming closed")
				break
			}

			pls.accept <- ps
		}
	}
}

// --------------------------------------------
// ProtocolStruct functions
//

//
// NewProtocol() takes a simple struct and creates a ProtocolStruct, pulls the Src and Dst from the context
// and initiates a connection to the defualt Dst address
// This function is called from the layer above to initalize the common protocol functions, and enable
// access to the standard functions provided by this layer.
//
func NewProtocol(ctx context.Context, s *simple.SimpleStruct) (ps *ProtocolStruct, e error) {

	//	contextlib.Logf(ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	dstID := contextlib.GetDest(ctx)
	if dstID == nil {
		contextlib.Logf(ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" GetDst() returned nil")
	}

	ps = NewProtocolStruct(ctx, s)

	return ps, e
}

func NewProtocolStruct(ctx context.Context, s *simple.SimpleStruct) (n *ProtocolStruct) {

	ctx, cancel := context.WithCancel(s.Ctx())
	eventchan := make(chan *simple.SimpleEvent)
	pcs := NewProtocolConnectionStruct(ctx, nil)
	pcs.SetState(ConnStateReady)

	n = &ProtocolStruct{
		ctx:       ctx,
		cancel:    cancel,
		simple:    s,
		eventchan: eventchan,
		srcconn:   pcs,
		dstconn: &ProtocolDstStruct{
			ctx:  ctx,
			conn: make(map[simple.ConnUniqueID]*ProtocolConnectionStruct),
		},
	}

	return n
}

//
// Ctx() returns the context of the ProtocolStruct
//
func (ps *ProtocolStruct) Ctx() context.Context {
	if ps == nil {
		panic(lumerinlib.FileLineFunc() + "ProtocolStruct is nil")
	}
	return ps.ctx
}

//
// Close()
//
func (ps *ProtocolStruct) Close() {

	// Perform orderly shutdown of all connection here

	ps.Cancel()
}

//
// Cancel() calls the simple layer Cancel function on the SimpleListenStruct
//
func (ps *ProtocolStruct) Cancel() {
	if ps == nil {
		panic(lumerinlib.FileLineFunc() + "ProtocolStruct is nil")
	}

	contextlib.Logf(ps.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	if ps.cancel == nil {
		contextlib.Logf(ps.ctx, contextlib.LevelError, fmt.Sprint(lumerinlib.FileLineFunc()+" cancel func is nil, struct:%v", ps))
		return
	}

	ps.dstconn.Cancel()
	ps.cancel()
}

//
// Run() calls the simple layer Run function on the SimpleListenStruct
//
func (ps *ProtocolStruct) Run() {
	if ps == nil {
		panic(lumerinlib.FileLineFunc() + "ProtocolStruct is nil")
	}
	//	contextlib.Logf(ps.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")
	ps.simple.Run()
}

//
//
//
func (ps *ProtocolStruct) GetSimpleEventChan() <-chan *simple.SimpleEvent {
	if ps == nil {
		panic(lumerinlib.FileLineFunc() + "ProtocolStruct is nil")
	}

	// contextlib.Logf(ps.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	c := ps.simple.GetEventChan()
	if c == nil {
		contextlib.Logf(ps.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" GetEVentChan returned nil, ps:%v", ps)
	}
	return c
}

//
// AsyncDial()
// opens a new connection to the desitnation
//
//
func (ps *ProtocolStruct) AsyncDial(dst *msgbus.Dest) (e error) {

	contextlib.Logf(ps.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	// Paranoid error checking here
	if ps == nil {
		panic(lumerinlib.FileLineFunc() + "ProtocolStruct is nil")
	}
	if ps.simple == nil {
		contextlib.Logf(ps.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" simple struct is nil")
	}
	if ps.simple.ConnectionStruct == nil {
		contextlib.Logf(ps.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" simple struct CoonectionStruct is nil")
	}

	e = ps.simple.AsyncDial(dst)

	return e
}

//
//
//
func (ps *ProtocolStruct) AsyncReDial(uid simple.ConnUniqueID) (e error) {
	e = ps.simple.AsyncReDial(uid)
	return e
}

//
// SetDefaultRouteUID()
// Set the SIMPL layer default route
//
func (ps *ProtocolStruct) SetDefaultRouteUID(uid simple.ConnUniqueID) (e error) {

	if uid < 0 {
		contextlib.Logf(ps.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" uid < 0")
	}

	defuid, _ := ps.GetDefaultRouteUID()

	if defuid == uid {
		contextlib.Logf(ps.ctx, contextlib.LevelWarn, lumerinlib.FileLineFunc()+" default route already set to UID: %d", uid)
	} else {
		e = ps.simple.SetRoute(uid)
	}

	return e
}

//
// GetDstStruct()
//
func (ps *ProtocolStruct) GetDstStruct() (pds *ProtocolDstStruct) {
	return ps.dstconn
}

//
// DstConn()
//
func (ps *ProtocolStruct) GetDstConn(uid simple.ConnUniqueID) (pcs *ProtocolConnectionStruct, e error) {
	if ps.dstconn.conn[uid] == nil {
		e = fmt.Errorf(lumerinlib.FileLineFunc()+"Index:%d does not exist", uid)
	} else {
		pcs = ps.dstconn.conn[uid]
	}
	return pcs, e
}

//
// SrcConn()
//
func (ps *ProtocolStruct) GetSrcConn() (pcs *ProtocolConnectionStruct, e error) {
	if ps.srcconn == nil {
		e = fmt.Errorf(lumerinlib.FileLineFunc() + "SrcConn does not exist yet")
	} else {
		pcs = ps.srcconn
	}
	return pcs, e
}

//
// GetDefaultRouteUID()
// get the  SIMPL layer default route
//
func (ps *ProtocolStruct) GetDefaultRouteUID() (uid simple.ConnUniqueID, e error) {
	uid, e = ps.simple.GetRoute()
	return uid, e
}

//
// Write()
// writes to the default route
//
func (ps *ProtocolStruct) Write(msg []byte) (count int, e error) {
	count = 0

	uid, e := ps.GetDefaultRouteUID()
	if e != nil {
		return 0, e
	}

	if uid < 0 {
		e = ErrDefaultRouteNotSet
		return 0, e
	}

	_, ok := ps.dstconn.conn[uid]
	if !ok {
		return 0, fmt.Errorf(lumerinlib.FileLineFunc() + "UID:%d not found in ProtocolConnectionStruct")
	}
	state := ps.dstconn.conn[uid].GetState()

	if ConnStateReady != state {
		e = fmt.Errorf(lumerinlib.FileLineFunc()+" Connection state:%s", state)
	} else {
		count, e = ps.simple.Write(uid, msg)
	}

	return count, e
}

//
// GetSrcRemoteAddr()
//
func (ps *ProtocolStruct) GetSrcRemoteAddr() (addr net.Addr, e error) {

	addr, e = ps.simple.GetRemoteAddrIdx(-1)

	return addr, e
}

//
// WriteSrc()
//
func (ps *ProtocolStruct) WriteSrc(msg []byte) (count int, e error) {

	count, e = ps.simple.Write(-1, msg)

	return count, e
}

//
// WriteDst()
//
func (ps *ProtocolStruct) WriteDst(index simple.ConnUniqueID, msg []byte) (count int, e error) {
	count, e = ps.simple.Write(index, msg)
	return count, e
}

//
// CloseDst()
//
func (ps *ProtocolStruct) CloseDst(index simple.ConnUniqueID) (e error) {
	return ps.simple.CloseConnection(index)
}

//
// CloseSrc()
//
func (ps *ProtocolStruct) CloseSrc() {
	ps.Close()
}

//
// Pub() publishes data, and stores the request ID to match the Completion Event
//
func (ps *ProtocolStruct) Pub(msgtype simple.MsgType, id simple.IDString, data interface{}) (rid int, e error) {

	rid, e = ps.simple.Pub(msgtype, id, data)
	return rid, e
}

//
//
//
func (ps *ProtocolStruct) Unpub(msgtype simple.MsgType, id simple.IDString) (rid int, e error) {

	rid, e = ps.simple.Unpub(msgtype, id)
	return rid, e
}

//
//
//
func (ps *ProtocolStruct) Sub(msgtype simple.MsgType, id simple.IDString) (rid int, e error) {

	rid, e = ps.simple.Sub(msgtype, id)
	return rid, e
}

//
//
//
func (ps *ProtocolStruct) Unsub(msgtype simple.MsgType, id simple.IDString) (rid int, e error) {

	rid, e = ps.simple.Unsub(msgtype, id)
	return rid, e
}

//
//
//
func (ps *ProtocolStruct) Get(msgtype simple.MsgType, id simple.IDString) (rid int, e error) {

	rid, e = ps.simple.Get(msgtype, id)
	return rid, e
}

//
//
//
func (ps *ProtocolStruct) Set(msgtype simple.MsgType, id simple.IDString, data interface{}) (rid int, e error) {

	rid, e = ps.simple.Set(msgtype, id, data)
	return rid, e
}

//
//
//
func (ps *ProtocolStruct) SearchIP(msgtype simple.MsgType, search string) (rid int, e error) {

	rid, e = ps.simple.SearchIP(msgtype, search)
	return rid, e
}

//
//
//
func (ps *ProtocolStruct) SearchMac(msgtype simple.MsgType, search string) (rid int, e error) {

	rid, e = ps.simple.SearchMac(msgtype, search)
	return rid, e
}

//
//
//
func (ps *ProtocolStruct) SearchName(msgtype simple.MsgType, search string) (rid int, e error) {

	rid, e = ps.simple.SearchName(msgtype, search)
	return rid, e
}
