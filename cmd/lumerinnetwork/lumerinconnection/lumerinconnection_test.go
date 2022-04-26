package lumerinconnection

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"testing"
	"time"

	"gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/sockettcp"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

var basePort int = 50000

type testAddr struct {
	ipaddr  string
	network string
}

//
// Open up a Listening port,
//
func TestSetupListenCancel(t *testing.T) {

	ctx := context.Background()

	cs := &contextlib.ContextStruct{}
	ctx = context.WithValue(ctx, contextlib.ContextKey, cs)

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)
	testaddr := &testAddr{
		network: "tcp",
		ipaddr:  addr,
	}

	l, e := testListen(ctx, testaddr)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Listen() Failed: %s\n", e))
	}

	l.Run()

	l.Cancel()

	select {
	case <-l.ctx.Done():
		e := ctx.Err()
		if e != nil {
			t.Fatal(lumerinlib.FileLine()+" CTX Done(): %s\n", ctx.Err())
		}

	case <-time.After(time.Second * 2):
		t.Fatal(lumerinlib.FileLine() + " select timeout ")

	case <-l.GetAcceptChan():
		t.Fatal(lumerinlib.FileLine() + " <-Accept() Returned, wtf")
	}

}

//
// Open up a Listening port, and close it
//
func TestSetupListenConnect(t *testing.T) {

	ctx := context.Background()

	cs := &contextlib.ContextStruct{}
	ctx = context.WithValue(ctx, contextlib.ContextKey, cs)

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)
	testaddr := &testAddr{
		network: "tcp",
		ipaddr:  addr,
	}

	l, e := testListen(ctx, testaddr)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Listen() Failed: %s\n", e))
	}

	l.Run()

	// Connect here
	s, e := sockettcp.Dial(ctx, "tcp", addr)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" sockettcp.Dial() Errored: %s\n", e))
	}
	if s == nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine() + " sockettcp.Dial() failed"))
	}

	t.Logf(lumerinlib.FileLine() + " Dial completed\n")

	select {
	case <-l.GetAcceptChan():
		t.Logf(lumerinlib.FileLine() + "GetAcceptChan() returned ok")

	case <-time.After(time.Second * 2):
		t.Fatal(lumerinlib.FileLine() + "Select Timeout")

	case <-l.ctx.Done():
		e := l.ctx.Err()
		if e != nil {
			t.Fatal(lumerinlib.FileLine()+" Listener CTX Done() Error: %s\n", e)
		}
		t.Fatal(lumerinlib.FileLine() + " Listener Done() Returned")

	case <-s.Ctx().Done():
		e := s.Ctx().Err()
		if e != nil {
			t.Fatal(lumerinlib.FileLine()+" socket CTX Done() Error: %s\n", e)
		}
		t.Fatal(lumerinlib.FileLine() + " Socket Done() Returned")

	}

}

//
//
//
func TestDialOut(t *testing.T) {

	ctx := context.Background()

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)
	testaddr := &testAddr{
		network: "tcp",
		ipaddr:  addr,
	}

	cs := &contextlib.ContextStruct{}
	ctx = context.WithValue(ctx, contextlib.ContextKey, cs)

	//
	// Open Listener
	//
	l, e := testListen(ctx, testaddr)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Listen() Failed: %s\n", e))
	}

	l.Run()

	testDial(ctx, testaddr)
	// s := testDial(ctx, testaddr)

	// Accept connection
	select {
	case <-l.GetAcceptChan():
	case <-time.After(time.Second * 1):
		t.Fatal(fmt.Errorf(lumerinlib.FileLine() + " timeout on Accept()"))
	}

}

//
//
//
func TestDialOutReadWrite(t *testing.T) {

	ctx := context.Background()
	var TestString = "This is a test string\n"
	var lsocket *LumerinSocketStruct

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)
	testaddr := &testAddr{
		network: "tcp",
		ipaddr:  addr,
	}

	cs := &contextlib.ContextStruct{}
	ctx = context.WithValue(ctx, contextlib.ContextKey, cs)

	//
	// Open Listener
	//
	l, e := testListen(ctx, testaddr)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Listen() Failed: %s\n", e))
	}

	l.Run()

	s := testDial(ctx, testaddr)

	select {
	case lsocket = <-l.GetAcceptChan():
	case <-time.After(time.Second * 1):
		t.Fatal(fmt.Errorf(lumerinlib.FileLine() + " timeout on Accept()"))
	}

	writeb := []byte(TestString)
	writecount, e := s.Write(writeb)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() Test Failed: %s\n", e))
	}
	if writecount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() Test Failed: %s\n", e))
	}

	fmt.Printf(lumerinlib.FileLine() + " Write() completed\n")

	readbuf := make([]byte, 64)
	readcount, e := lsocket.Read(readbuf)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" RedBytes() Test Failed: %s\n", e))
	}
	if readcount != writecount {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+"Count Test Failed read: %d, write: %d\n", readcount, writecount))
	}

}

// ---------------------------------------------------------------------------------

//
// testListen()
// Dont forget to call Run()
//
func testListen(ctx context.Context, addr net.Addr) (l *LumerinListenStruct, e error) {
	return NewListen(ctx, addr)
}

//
//
//
// func testDial(ctx context.Context, network LumProto, port int, ip net.IPAddr) (s *LumerinSocketStruct) {
func testDial(ctx context.Context, addr net.Addr) (s *LumerinSocketStruct) {

	s, e := Dial(ctx, addr)
	if e != nil {
		fmt.Printf(lumerinlib.FileLine()+" Dial Test Failed: %s\n", e)
		panic(fmt.Sprintf(lumerinlib.FileLine()+" Dial Test Failed: %s\n", e))
	}

	return s
}

//
//
//
func goTestAcceptChannelEcho(l *LumerinListenStruct) {

	fmt.Printf(lumerinlib.FileLine() + " Waiting on Connection\n")

	s := <-l.GetAcceptChan()

	if s == nil {
		panic(fmt.Sprintf(lumerinlib.FileLine() + " scoket is nil"))
	}

	fmt.Printf(lumerinlib.FileLine() + " Connection Accepted\n")

	for {
		buf := make([]byte, 2)
		fmt.Printf(lumerinlib.FileLine() + " Read()ing\n")
		readcount, e := s.Read(buf)
		if e == io.EOF {
			fmt.Printf(lumerinlib.FileLine()+" Read() EOF count:%d\n", readcount)
			return
		}
		if e != nil {
			panic(fmt.Sprintf(lumerinlib.FileLine()+" Read Failed: %s\n", e))
		}

		if readcount != 0 {
			buf = buf[:readcount]
			writecount, e := s.Write(buf)
			if e != nil {
				panic(fmt.Sprintf(lumerinlib.FileLine()+" write Failed: %s\n", e))
			}
			if writecount == 0 {
				panic(fmt.Sprintf(lumerinlib.FileLine() + " write Failed: Zero bytes written\n"))
			}
		} else {
			panic(fmt.Sprintf(lumerinlib.FileLine() + " readcount == 0"))
		}
	}
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
