package protocol

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"

	simple "gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/SIMPL"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

var ErrProtoConnNoDstIndex = errors.New("DST index not found")
var ErrProtoConnNoID = errors.New("Connection ID not found")

type ConnectionState string

const ConnStateNew ConnectionState = "New"
const ConnStateInit ConnectionState = "Init"
const ConnStateReady ConnectionState = "Ready"
const ConnStateError ConnectionState = "Error"

//
// Protocol encapulates the non upper layer protocol specific items
// such as the connection management, which should be the same for all
// It also encapsulates the management functions and tracking of outstanding
// Actions waiting on events
//
// Communications, MsgBus
//
//

// Addr stores the address of the connection
// State shows the current state of the conneciton
// UID is the unique ID from the SIMPL layer
// Err shows the error state if it is in the error state
type ProtocolConnectionStruct struct {
	ctx    context.Context
	cancel func()
	dest   *msgbus.Dest
	state  ConnectionState
	err    error
	buffer []byte
}

type ProtocolDstStruct struct {
	ctx  context.Context
	conn map[simple.ConnUniqueID]*ProtocolConnectionStruct
}

//
//
//
func NewProtocolConnectionStruct(ctx context.Context, dest *msgbus.Dest) (pcs *ProtocolConnectionStruct) {

	//	contextlib.Logf(ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	ctx, cancel := context.WithCancel(ctx)
	pcs = &ProtocolConnectionStruct{
		ctx:    ctx,
		cancel: cancel,
		dest:   dest,
		state:  ConnStateNew,
		err:    nil,
		buffer: make([]byte, 0, 1024),
	}

	return pcs
}

//
// New() finds or creates an open slot and starts to open a connection to the dst address
//
func (p *ProtocolDstStruct) NewProtocolDstStruct(osce *simple.SimpleConnOpenEvent) (e error) {

	contextlib.Logf(p.ctx, contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	_, exists := p.conn[osce.UniqueID()]
	if exists {
		contextlib.Logf(p.ctx, contextlib.LevelInfo, lumerinlib.FileLineFunc()+" Dst UID:%d exsists", osce.UniqueID())
		// e = fmt.Errorf(lumerinlib.FileLineFunc()+" UniqueID:%d is already used", osce.UniqueID())
	}

	p.conn[osce.UniqueID()] = NewProtocolConnectionStruct(p.ctx, osce.Dest())

	return e
}

//
// These should use an Interface.... circle back to this later
//

// ---------------------------------------------------
// *ProtocolDstStruct()

//
// Cancel()
//
func (p *ProtocolDstStruct) Cancel() (e error) {

	// Close all connections
	for uid, _ := range p.conn {
		if p.conn[uid] != nil {
			p.conn[uid].Cancel()
		} else {
			e = fmt.Errorf(lumerinlib.FileLineFunc()+"Index:%d does not exist", uid)
			break
		}
	}

	return e
}

//
// CancelUID()
//
func (p *ProtocolDstStruct) CancelUID(uid simple.ConnUniqueID) (e error) {
	if p.conn[uid] == nil {
		e = fmt.Errorf(lumerinlib.FileLineFunc()+"Index:%d does not exist", uid)
	} else {
		p.conn[uid].Cancel()
	}

	return e
}

//
// GetAddr()
//
func (p *ProtocolDstStruct) GetAddr(uid simple.ConnUniqueID) (addr net.Addr, e error) {
	if p.conn[uid] == nil {
		e = fmt.Errorf(lumerinlib.FileLineFunc()+"Index:%d does not exist", uid)
	} else {
		addr, e = p.conn[uid].dest.NetAddr()
	}

	return addr, e
}

//
// GetState()
//
func (p *ProtocolDstStruct) GetState(uid simple.ConnUniqueID) (state ConnectionState, e error) {
	if p.conn[uid] == nil {
		e = fmt.Errorf(lumerinlib.FileLineFunc()+"Index:%d does not exist", uid)
	} else {
		state = p.conn[uid].state
	}

	return state, e
}

//
// SetState()
//
func (p *ProtocolDstStruct) SetState(uid simple.ConnUniqueID, s ConnectionState) (e error) {
	if p.conn[uid] == nil {
		e = fmt.Errorf(lumerinlib.FileLineFunc()+"Index:%d does not exist", uid)
	} else {
		p.conn[uid].state = s
	}

	return e
}

//
//
//
func (p *ProtocolDstStruct) GetError(uid simple.ConnUniqueID) (err error, e error) {
	if p.conn[uid] == nil {
		e = fmt.Errorf(lumerinlib.FileLineFunc()+"Index:%d does not exist", uid)
	} else {
		err = p.conn[uid].err
	}

	return err, e
}

//
//
//
func (p *ProtocolDstStruct) SetError(uid simple.ConnUniqueID, err error) (e error) {
	if p.conn[uid] == nil {
		e = fmt.Errorf(lumerinlib.FileLineFunc()+"Index:%d does not exist", uid)
	} else {
		p.conn[uid].err = err
	}

	return e
}

// ---------------------------------------------------
// *ProtocolConnectionStruct)

//
// Ctx()
//
func (p *ProtocolConnectionStruct) Ctx() context.Context {
	return p.ctx
}

//
// Cancel()
//
func (p *ProtocolConnectionStruct) Cancel() {
	p.cancel()
}

//
// GetAddr()
//
func (p *ProtocolConnectionStruct) GetAddr() (addr net.Addr) {
	addr, _ = p.dest.NetAddr()
	return addr
}

//
// GetState()
//
func (p *ProtocolConnectionStruct) GetState() ConnectionState {
	return p.state
}

//
// SetState()
//
func (p *ProtocolConnectionStruct) SetState(s ConnectionState) {
	p.state = s
}

//
//
//
func (p *ProtocolConnectionStruct) GetError() error {
	return p.err
}

//
//
//
func (p *ProtocolConnectionStruct) SetError(e error) {
	p.err = e
}

//
//
//
func (p *ProtocolConnectionStruct) AddBuf(buf []byte) {
	if p == nil {
		panic(lumerinlib.FileLineFunc())
	}
	p.buffer = append(p.buffer, buf...)
}

//
//
//
func (p *ProtocolConnectionStruct) GetLineTermData() (buf []byte, e error) {
	if p == nil {
		panic(lumerinlib.FileLineFunc())
	}

	buf = make([]byte, 0)
	if len(p.buffer) > 0 {

		// idx = -1, no data, idx = 0, 1 byte ('\n')
		idx := bytes.Index(p.buffer, []byte("\n"))

		if idx == 0 {
			p.buffer = p.buffer[1:]
		} else if idx > 0 {
			idx++
			buf = append(buf, p.buffer[:idx]...)
			p.buffer = p.buffer[idx:]
		}
	}
	return buf, e
}
