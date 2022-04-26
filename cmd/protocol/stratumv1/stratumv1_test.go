package stratumv1

import (
	"context"
	"fmt"
	"net"
	"testing"

	"gitlab.com/TitanInd/lumerin/cmd/log"
	simple "gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/SIMPL"
	"gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/sockettcp"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/protocol"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
	"gitlab.com/TitanInd/lumerin/lumerinlib/testinglib"
)

var basePort int = 50000
var localhost string = "127.0.0.1"

// var testAuthorizeMsg = `{"id":2,"method":"mining.authorize","params":["testrig.worker1",""]}`

var defaultTitanDest = createDefaultDest("stratum+tcp://sean.worker:@mining.dev.pool.titan.io:4242/")

//
// The basics, open up a listening connection...
//
func TestNewProto(t *testing.T) {

	localport := testinglib.GetRandPort()
	listenerstr := fmt.Sprintf("%s:%d", localhost, localport)
	ctx := newContextStruct(t, listenerstr, defaultTitanDest)

	sls := newStratumConnection(t, ctx)
	sls.Run()
	sls.Cancel()

}

//
// Setup Stratum listener, and make a TCP connection to it.
//
func TestNewStratumConnection(t *testing.T) {

	localport := testinglib.GetRandPort()
	addr := fmt.Sprintf("%s:%d", localhost, localport)
	ctx := newContextStruct(t, addr, defaultTitanDest)

	sls := newStratumConnection(t, ctx)
	sls.Run()

	// connect to listening port with socket
	s, e := sockettcp.Dial(ctx, "tcp", addr)
	if e != nil {
		t.Errorf("sockettcp.Dial() returned error:%s", e)
	}
	if s == nil {
		t.Errorf("sockettcp.Dial() returned nil")
	}

	sls.Cancel()
}

//
//
//
func TestNewSrc2PoolConnectionsOpened(t *testing.T) {

	var event *msgbus.Event
	var err error

	localport := testinglib.GetRandPort()
	localpoolport1 := testinglib.GetRandPort()
	localpoolport2 := testinglib.GetRandPort()
	nodeaddr := fmt.Sprintf("%s:%d", localhost, localport)
	pooladdr1 := fmt.Sprintf("%s:%d", localhost, localpoolport1)
	pooladdr2 := fmt.Sprintf("%s:%d", localhost, localpoolport2)
	poolurl1 := fmt.Sprintf("stratum+tcp://username:password@%s:%d", localhost, localpoolport1)
	poolurl2 := fmt.Sprintf("stratum+tcp://username:password@%s:%d", localhost, localpoolport2)

	defdestid := msgbus.IDString("LocalPriPoolDestID")
	secdestid := msgbus.IDString("LocalSecPoolDestID")
	defdest := createDest(defdestid, poolurl1)
	secdest := createDest(secdestid, poolurl2)
	_ = secdest

	ctx := newContextStruct(t, nodeaddr, defdest)

	event, err = contextlib.GetContextStruct(ctx).MsgBus.PubWait(msgbus.DestMsg, secdestid, secdest)
	if err != nil {
		t.Fatalf(fmt.Sprintf("Adding Default Dest Failed: %s", err))
	}
	if event.Err != nil {
		t.Fatalf(fmt.Sprintf("Adding Default Dest Failed: %s", event.Err))
	}

	// ------------------------------------
	// Verify DefaultDest record in MsgBus
	// ------------------------------------

	event, err = contextlib.GetContextStruct(ctx).MsgBus.GetWait(msgbus.DestMsg, defdestid)
	if err != nil {
		t.Fatalf("GetWait() failed %s", err)
	}
	if event.Err != nil {
		t.Fatalf("GetWait() failed %s", event.Err)
	}

	// --------------------------------------
	// Run the Node that we are testing
	// --------------------------------------
	sls := newStratumConnection(t, ctx)
	sls.Run()

	// ------------------------------------
	// Pool1 Listener Setup
	// ------------------------------------
	listener1, e := socketListener(t, ctx, pooladdr1)
	if e != nil {
		t.Fatalf("socketListener failed: %s", e)
	}

	if listener1.Done() {
		t.Fatalf("fakelistener is closed")
	}

	// ------------------------------------
	// Pool2 Listener Setup
	// ------------------------------------
	listener2, e := socketListener(t, ctx, pooladdr2)
	if e != nil {
		t.Fatalf("socketListener failed: %s", e)
	}

	if listener2.Done() {
		t.Fatalf("fakelistener is closed")
	}

	// ---------------------------------------
	// Trigger a new instance of Stratum
	// Socket dial to the local node addr
	// SRC connection (miner)
	// ---------------------------------------
	stratumsrc, e := sockettcp.Dial(ctx, "tcp", nodeaddr)
	if e != nil {
		t.Errorf("sockettcp.Dial() returned error:%s", e)
	}
	if stratumsrc == nil {
		t.Errorf("sockettcp.Dial() returned nil")
	}

	// --------------------------------------
	// Expect a connection here in pool1
	// as a result of the socket connection
	// --------------------------------------
	var poolsocket1 *sockettcp.SocketTCPStruct
	poolsocketinterface1 := <-listener1.GetAcceptChan()

	switch poolsocketinterface1.(type) {
	case *sockettcp.SocketTCPStruct:
		poolsocket1 = poolsocketinterface1.(*sockettcp.SocketTCPStruct)
	default:
		t.Fatalf("poolsocketinterface type:%T", poolsocketinterface1)
	}
	_ = poolsocket1

	// --------------------------------------
	// Pool1 Read
	// Read looking for Subscribe message
	// --------------------------------------
	read1 := make([]byte, 1024)
	count, e := poolsocket1.Read(read1)
	if e != nil {
		t.Errorf("Read() returned error:%s", e)
	}
	t.Logf("Read():%s", read1[:count])

	// ---------------------------------------
	// SEND Subscribe Request to the Node
	// {"id": 1, "method": "mining.subscribe", "params": ["cpuminer/2.5.1"]}
	// ---------------------------------------
	subscribeReq := &stratumRequest{
		ID:     0,
		Method: string(CLIENT_MINING_SUBSCRIBE),
	}
	subscribeReq.Params = append(subscribeReq.Params, "GoLangTestMiner/0.99")
	msg, e := subscribeReq.createRequestMsg()
	if e != nil {
		t.Fatalf("Subscribe createRequestMsg() error:%s", e)
	}
	count, e = stratumsrc.Write(msg)
	if e != nil {
		t.Fatalf("Write() error:%s", e)
	}
	if count != len(msg) {
		t.Fatalf("Write() count != len:%d, %d", count, len(msg))
	}

	// ---------------------------------------
	// Read Subscribe Response (parse later)
	// ---------------------------------------
	stratumReadBuf := make([]byte, 1024)
	count, e = stratumsrc.Read(stratumReadBuf)
	if e != nil {
		t.Fatalf("Write() error:%s", e)
	}
	t.Logf("Read():%s", stratumReadBuf[:count])

	// ---------------------------------------
	// Send Authorize Request
	// {"id": 2, "method": "mining.authorize", "params": ["username.worker0", ""]}
	// ---------------------------------------
	authReq := &stratumRequest{
		ID:     0,
		Method: string(CLIENT_MINING_AUTHORIZE),
	}
	authReq.Params = append(authReq.Params, "username.Worker0")
	msg, e = authReq.createRequestMsg()
	if e != nil {
		t.Fatalf("Auth createRequestMsg() error:%s", e)
	}
	count, e = stratumsrc.Write(msg)
	if e != nil {
		t.Fatalf("Write() error:%s", e)
	}
	if count != len(msg) {
		t.Fatalf("Write() count != len:%d, %d", count, len(msg))
	}

	// ---------------------------------------
	// Read Auth Response (parse later)
	// ---------------------------------------
	stratumReadBuf = make([]byte, 1024)
	count, e = stratumsrc.Read(stratumReadBuf)
	if e != nil {
		t.Fatalf("Write() error:%s", e)
	}
	t.Logf("Read():%s", stratumReadBuf[:count])

	// ----------------------------------------
	// Get the miner record
	// Update the miner record with the new pool2 dest
	// to trigger the destination change
	// causing a new connecton to pool2
	// ----------------------------------------

	// Get ALL miner records
	event, err = contextlib.GetContextStruct(ctx).MsgBus.GetWait(msgbus.MinerMsg, "")
	if err != nil {
		t.Fatalf("GetWait() failed %s", err)
	}
	if event.Err != nil {
		t.Fatalf("GetWait() failed %s", event.Err)
	}

	if event.EventType != msgbus.GetIndexEvent {
		t.Fatalf("GetWait() failed to return an index")
	}

	// Get The First miner record
	minerRecID := event.Data.(msgbus.IDIndex)[0]
	event, err = contextlib.GetContextStruct(ctx).MsgBus.GetWait(msgbus.MinerMsg, minerRecID)
	if err != nil {
		t.Fatalf("GetWait() failed %s", err)
	}
	if event.Err != nil {
		t.Fatalf("GetWait() failed %s", event.Err)
	}

	var minerRec msgbus.Miner
	switch event.Data.(type) {
	case msgbus.Miner:
		minerRec = event.Data.(msgbus.Miner)
	case *msgbus.Miner:
		minerRec = *event.Data.(*msgbus.Miner)
	default:
		t.Fatalf("event.Data is not msgbus.Miner %v", event)
	}

	// the destinations should be different at this point
	if minerRec.Dest != defdest.ID {
		t.Fatalf("minerRec.ID is not the default dest ID %v", event)
	}

	// Push the updated destination to the miner record,
	// triggering an opening of the second connection to pool2
	minerRec.Dest = secdest.ID
	event, err = contextlib.GetContextStruct(ctx).MsgBus.SetWait(msgbus.MinerMsg, minerRecID, &minerRec)
	if err != nil {
		t.Fatalf("GetWait() failed %s", err)
	}
	if event.Err != nil {
		t.Fatalf("PubWait() failed %s", event.Err)
	}

	//
	// This is the Sec Dest pool2,
	// so switching destinations should result in a conntion being opened here.
	//
	var poolsocket2 *sockettcp.SocketTCPStruct
	poolsocketinterface2 := <-listener2.GetAcceptChan()

	switch poolsocketinterface2.(type) {
	case *sockettcp.SocketTCPStruct:
		poolsocket2 = poolsocketinterface2.(*sockettcp.SocketTCPStruct)
	default:
		t.Fatalf("poolsocketinterface type:%T", poolsocketinterface2)
	}
	_ = poolsocket2

	// Should see a subscribe message
	read2 := make([]byte, 1024)
	count, e = poolsocket2.Read(read2)
	if e != nil {
		t.Errorf("Read() returned error:%s", e)
	}
	t.Logf("Read():%s", read2[:count])

	//
	// Send Subscribe Response
	//{"id":1,"error":null,"result":[[["mining.notify","0"]],"1",1]}
	//
	response := &stratumResponse{}
	msg, e = response.createSrcSubscribeResponseMsg(1)
	if e != nil {
		t.Fatalf("Subscribe createRequestMsg() error:%s", e)
	}
	count, e = poolsocket2.Write(msg)
	if e != nil {
		t.Fatalf("Write() error:%s", e)
	}
	if count != len(msg) {
		t.Fatalf("Write() count != len:%d, %d", count, len(msg))
	}

	// Should see a subscribe message
	read2 = make([]byte, 1024)
	count, e = poolsocket2.Read(read2)
	if e != nil {
		t.Errorf("Read() returned error:%s", e)
	}
	t.Logf("Read():%s", read2[:count])

	//
	// Send Auth Response
	//{"id":2,"error":null,"result":true}
	//
	response.ID = 0
	response.Result = true
	response.Error = nil
	msg, e = response.createResponseMsg()
	if e != nil {
		t.Fatalf("Subscribe createResponseMsg() error:%s", e)
	}
	count, e = poolsocket2.Write(msg)
	if e != nil {
		t.Fatalf("Write() error:%s", e)
	}
	if count != len(msg) {
		t.Fatalf("Write() count != len:%d, %d", count, len(msg))
	}

	//
	// Send Set Difficulty
	// {"id":0,"method":"mining.set_difficulty","params":["65535.000000"]}
	//
	setDiffReq := &stratumRequest{
		ID:     1500,
		Method: string(SERVER_MINING_SET_DIFFICULTY),
	}
	setDiffReq.Params = append(setDiffReq.Params, "65000.00001")
	msg, e = setDiffReq.createRequestMsg()
	if e != nil {
		t.Fatalf("Subscribe createRequestMsg() error:%s", e)
	}
	count, e = poolsocket2.Write(msg)
	if e != nil {
		t.Fatalf("Write() error:%s", e)
	}
	if count != len(msg) {
		t.Fatalf("Write() count != len:%d, %d", count, len(msg))
	}

	// ---------------------------------------
	// Read Set Difficulty Request
	// ---------------------------------------
	stratumReadBuf = make([]byte, 1024)
	count, e = stratumsrc.Read(stratumReadBuf)
	if e != nil {
		t.Fatalf("Write() error:%s", e)
	}
	t.Logf("Read():%s", stratumReadBuf[:count])

	// ---------------------------------------
	// Read Set Difficulty Request
	// ---------------------------------------
	stratumReadBuf = make([]byte, 1024)
	count, e = stratumsrc.Read(stratumReadBuf)
	if e != nil {
		t.Fatalf("Write() error:%s", e)
	}
	t.Logf("Read():%s", stratumReadBuf[:count])

	// <-sls.Ctx().Done()
	//t.Logf("Geeting ready to leave")
	sls.Cancel()

}

// ---------------------------------------------------------------------------
//
//
func newContextStruct(t *testing.T, srcstr string, dest *msgbus.Dest) (ret context.Context) {

	ctx := context.Background()

	l := log.New()
	mb := msgbus.New(10, l)

	cs := &contextlib.ContextStruct{}

	cs.SetLog(l)
	cs.SetMsgBus(mb)

	srcaddr, e := net.ResolveTCPAddr("tcp", srcstr)
	if e != nil {
		t.Fatalf(fmt.Sprintf("ResolveTCPAddr() error:%s", e))
	}
	cs.SetSrc(srcaddr)

	if srcstr != "" {
		src := lumerinlib.NewNetAddr(lumerinlib.TCP, srcstr)
		cs.SetSrc(src)
	}

	cs.SetDest(dest)
	validate_dest := cs.GetDest()

	if dest.ID != validate_dest.ID {
		t.Fatalf(fmt.Sprintf("Retrieveing Dest Failed"))
	}

	validate_mb := cs.GetMsgBus()

	event, err := validate_mb.PubWait(msgbus.DestMsg, msgbus.IDString(dest.ID), dest)
	if err != nil {
		t.Fatalf(fmt.Sprintf("Adding Default Dest Failed: %s", err))
	}
	if event.Err != nil {
		t.Fatalf(fmt.Sprintf("Adding Default Dest Failed: %s", event.Err))
	}

	event, err = validate_mb.GetWait(msgbus.DestMsg, msgbus.IDString(dest.ID))
	if err != nil {
		t.Fatalf(fmt.Sprintf("Retrieving Default Dest Failed: %s", err))
	}
	if event.Err != nil {
		t.Fatalf(fmt.Sprintf("Retrieveing Default Dest Failed: %s", event.Err))
	}

	ret = contextlib.SetContextStruct(ctx, cs)
	return ret
}

//
// newConnection()
//
func newStratumConnection(t *testing.T, ctx context.Context) (sls *StratumV1ListenStruct) {

	cs := contextlib.GetContextStruct(ctx)
	srcaddr := cs.GetSrc()
	dst := cs.GetDest()

	sls, err := NewListener(ctx, srcaddr, dst)

	sls.Run()

	if err != nil {
		t.Errorf("NewListner() returned error:%s", err)
	}
	if sls == nil {
		t.Errorf("NewListner() returned nil")

	}

	return sls
}

//
//
//
func connect(t *testing.T, ctx context.Context, addr string) (s *sockettcp.SocketTCPStruct, e error) {
	s, e = sockettcp.Dial(ctx, "tcp", addr)
	if e != nil {
		t.Errorf("Dial() returned error:%s", e)
	}

	return s, e
}

//
//
//
func testNewStratumV1(ss *simple.SimpleStruct) {

	if ss == nil {
		panic(lumerinlib.FileLineFunc() + " nil SimpleStruct")
	}
	if ss.ConnectionStruct == nil {
		panic(lumerinlib.FileLineFunc() + " nil SimpleStruct.ConnectionStruct")
	}

	contextlib.Logf(ss.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	cs := contextlib.GetContextStruct(ss.Ctx())

	if cs == nil {
		contextlib.Logf(ss.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Context Structre not correct")
	}

	dst := cs.GetDest()
	if dst == nil {
		contextlib.Logf(ss.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Context Struct DST not defined")
	}

	// inialize a new ProtocolStruct to gain access to the standard protocol functions
	// The default Dst should be opened when this returns
	pls, err := protocol.NewProtocol(ss.Ctx(), ss)
	if err != nil {
		contextlib.Logf(ss.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Create NewProtocol() failed: %s", err)
	}
	if pls == nil {
		contextlib.Logf(ss.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Create NewProtocol() failed - no pointer returned")
	}

	svs := &StratumV1Struct{
		protocol:            pls,
		minerRec:            nil,
		srcSubscribeRequest: nil,
		srcAuthRequest:      nil,
		// Fill in other state information here
	}

	// Launch the event handler
	go svs.goEvent()

	ss.Run()

}

//
//
//
func socketListener(t *testing.T, ctx context.Context, listenaddr string) (l *sockettcp.ListenTCPStruct, e error) {

	l, e = sockettcp.NewListen(ctx, "tcp", string(listenaddr))
	if e != nil {
		t.Fatalf("NewListen() returned error:%s", e)
	}
	if l == nil {
		t.Fatalf("NewListen() returned nil - error:%s", e)
	}

	l.Run()

	return l, e
}

//
//
//
func createDefaultDest(poolurl string) (dest *msgbus.Dest) {

	dest = &msgbus.Dest{
		ID:     msgbus.DestID(msgbus.DEFAULT_DEST_ID),
		NetUrl: msgbus.DestNetUrl(poolurl),
	}
	return dest
}

//
//
//
func createDest(id msgbus.IDString, poolurl string) (dest *msgbus.Dest) {

	dest = &msgbus.Dest{
		ID:     msgbus.DestID(id),
		NetUrl: msgbus.DestNetUrl(poolurl),
	}
	return dest
}
