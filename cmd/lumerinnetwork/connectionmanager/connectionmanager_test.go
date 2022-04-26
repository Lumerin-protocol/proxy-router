package connectionmanager

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/lumerinconnection"
	"gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/sockettcp"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

var basePort int = 50000

var TestString = "This is a test string\n"

//
// Open up listening socket
//
func TestSetupListenCancel(t *testing.T) {

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)
	testaddr := &testAddr{
		network: "tcp",
		ipaddr:  addr,
	}

	ctx := context.Background()
	cs := &contextlib.ContextStruct{}
	cs.SetSrc(testaddr)
	ctx = context.WithValue(ctx, contextlib.ContextKey, cs)

	l, e := NewListen(ctx)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" Listen() Failed: %s\n", e))
	}

	l.Run()

	l.Cancel()

	select {
	case <-l.ctx.Done():
		t.Logf(lumerinlib.FileLineFunc()+" CTX Done(): %s\n", ctx.Err())
	case <-l.Accept():
		t.Logf(lumerinlib.FileLineFunc() + "Accept() OK: returned error:")
	case <-time.After(time.Second * 1):
		t.Fatal(fmt.Errorf(lumerinlib.FileLine() + " timeout on Accept()"))
	}

}

//
// Open up listener
//
//
func TestSrcDial(t *testing.T) {

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)
	testaddr := &testAddr{
		network: "tcp",
		ipaddr:  addr,
	}

	ctx := context.Background()
	cs := &contextlib.ContextStruct{}
	cs.SetLog(log.New())
	cs.SetSrc(testaddr)
	ctx = context.WithValue(ctx, contextlib.ContextKey, cs)

	cls, e := NewListen(ctx)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" Listen() Failed: %s\n", e))
	}

	cls.Run()

	// Setsup to Echo data sent to SRC channel back to the originator
	go goTestAcceptSrcChannelEcho(cls)

	//
	// Dial and write test data, recieve same test data
	//
	s, e := sockettcp.Dial(ctx, "tcp", addr)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" Dial() Failed: %s\n", e))
	}

	// Write into socket, read back from socket -- the data is echoed through the original connection

	fmt.Printf(lumerinlib.FileLineFunc() + " Dial completed\n")

	writeb := []byte(TestString)
	writecount, e := s.Write(writeb)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" Write() Test Failed: %s\n", e))
	}
	if writecount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" Write() Test Failed: %s\n", e))
	}

	fmt.Printf(lumerinlib.FileLineFunc() + " Write() completed\n")

	buf := make([]byte, 1024)
	count, e := s.Read(buf)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" ReadBytes() Test Failed: %s\n", e))
	}
	if count != len(TestString) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+"Count Test Failed read: %d, write: %d\n", count, len(TestString)))
	}

}

//
//
//
func TestSrcDefDstDial(t *testing.T) {

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)
	testaddr := &testAddr{
		network: "tcp",
		ipaddr:  addr,
	}

	ctx := context.Background()
	cs := &contextlib.ContextStruct{}
	cs.SetLog(log.New())
	cs.SetSrc(testaddr)
	ctx = context.WithValue(ctx, contextlib.ContextKey, cs)

	//
	// Setup Connection Manager listener
	//
	l, e := NewListen(ctx)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" Listen() Failed: %s\n", e))
	}

	l.Run()
	defer l.Close()

	//
	// Setup incoming connection echo mechanism
	//
	s := testSetupEchoConnection(t, l)
	defer s.Close()

	//
	// Write TestString into socket
	//
	writeb := []byte(TestString)
	writecount, e := s.DstWrite(writeb)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" DstWrite() Test Failed: %s\n", e))
	}
	if writecount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" DstWrite() Test Failed: %s\n", e))
	}

	fmt.Printf(lumerinlib.FileLineFunc() + " DstWrite() completed\n")

	soc, e := s.DstGetSocket()
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" DstGetSocket() Test Failed: %s\n", e))
	}

	fmt.Printf(lumerinlib.FileLineFunc() + " DstGetSocket() completed\n")

	readb := make([]byte, 64)
	readcount, e := soc.Read(readb)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" soc.Read() Error: %s\n", e))
	}
	if readcount != writecount {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+"Count Test Failed read: %d, write: %d\n", readcount, writecount))
	}

}

//
//
//
func TestSrcIdxDstDial(t *testing.T) {

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)
	testaddr := &testAddr{
		network: "tcp",
		ipaddr:  addr,
	}

	ctx := context.Background()
	cs := &contextlib.ContextStruct{}
	cs.SetLog(log.New())
	cs.SetSrc(testaddr)
	ctx = context.WithValue(ctx, contextlib.ContextKey, cs)

	l, e := NewListen(ctx)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" Listen() Failed: %s\n", e))
	}

	l.Run()

	defer l.Close()

	s := testSetupEchoConnection(t, l)

	defer s.Close()

	writeb := []byte(TestString)
	writecount, e := s.IdxWrite(0, writeb)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" IdxWrite() Test Failed: %s\n", e))
	}
	if writecount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" IdxWrite() Test Failed: %s\n", e))
	}

	fmt.Printf(lumerinlib.FileLineFunc() + " IdxWrite() completed\n")

	soc, e := s.IdxGetSocket(0)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" DstGetSocket() Test Failed: %s\n", e))
	}
	defer soc.Close()
	reader := bufio.NewReader(soc)
	readbuf, e := reader.ReadBytes('\n')
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" ReadBytes() Test Failed: %s\n", e))
	}
	readcount := len(readbuf)
	if readcount != writecount {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+"Count Test Failed read: %d, write: %d\n", readcount, writecount))
	}

}

// ---------------------------------------------------------------------------------------------------

//
// Create a connection to Connection Manager, and set the connection manager to echo back the data
//
func testSetupEchoConnection(t *testing.T, l *ConnectionListenStruct) (cs *ConnectionStruct) {

	fmt.Printf(lumerinlib.FileLineFunc() + " Waiting on Connection\n")

	lss, e := lumerinconnection.Dial(l.ctx, l.addr)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc()+" Dial() Failed: %s\n", e))
	}

	cs = <-l.Accept()
	if cs == nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLineFunc() + " Accept() returned nil"))
	}

	fmt.Printf(lumerinlib.FileLineFunc() + " Connection Accepted\n")

	cs.dst[0] = lss
	cs.defidx = 0

	go cs.goSrcChannelEcho()

	return cs
}

//
//
//
func goTestAcceptSrcChannelEcho(l *ConnectionListenStruct) {

	fmt.Printf(lumerinlib.FileLineFunc() + " Waiting on Connection\n")

	cs := <-l.Accept()

	if cs == nil {
		fmt.Printf(lumerinlib.FileLineFunc() + " ERROR: Socket Accept() returned nil\n")
		l.Close()
		return
	}

	fmt.Printf(lumerinlib.FileLineFunc() + " Connection Accepted\n")

	go cs.goSrcChannelEcho()
}

//
// One shot - Echo Read event data back to the SRC socket
// call Cancel if there is an error
//
func (cs *ConnectionStruct) goSrcChannelEcho() {

	fmt.Printf(lumerinlib.FileLineFunc() + " SRC Echo\n")

	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-time.After(time.Second * 5):
			return
		case readevent := <-cs.readChan:
			if readevent == nil {
				return
			}
			if readevent.index != -1 {
				fmt.Printf(lumerinlib.FileLineFunc()+" readChan incorrect index:%d\n", readevent.index)
				cs.Cancel()
				return
			}
			if readevent.count == 0 {
				fmt.Printf(lumerinlib.FileLineFunc() + " readevent count is 0")
				cs.Cancel()
				return
			}

			count, e := cs.SrcWrite(readevent.data)
			if e != nil {
				fmt.Printf(lumerinlib.FileLineFunc()+" SrcWrite() Returned Error:%s\n", e)
				cs.Cancel()
				return
			}
			if count != readevent.count {
				fmt.Printf(lumerinlib.FileLineFunc()+" SrcWrite() count does not match:%d, %d\n", count, readevent.count)
				cs.Cancel()
				return
			}
		}

	}
}

type testAddr struct {
	ipaddr  string
	network string
}

func (t *testAddr) Network() string {
	return t.network
}

func (t *testAddr) String() string {
	return t.ipaddr
}

//
//
//
func getRandPort() (port int) {
	port = rand.Intn(10000) + basePort
	return port
}
