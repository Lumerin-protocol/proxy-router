package sockettcp

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

var basePort int = 50000
var TestString = "This is the the test string\n"

func TestTCPSetupTestCancel(t *testing.T) {

	ctx, _ := contextlib.CreateNewContext(context.Background())

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)

	l, e := NewListen(ctx, "tcp", addr)
	if e != nil {
		t.Fatalf(fmt.Sprintf(lumerinlib.FileLine()+"Listen() Test Failed: %s", e))
	}

	l.Run()

	l.Cancel()

	select {
	case s := <-l.GetAcceptChan():
		if s != nil {
			t.Fatalf(fmt.Sprintf(lumerinlib.FileLine()+"Cancel() Test Failed:%v", s))
		}
	case <-l.ctx.Done():
		t.Logf(fmt.Sprintf(lumerinlib.FileLine() + "Cancel() Passed"))
	}

}

//
//
//
func TestTCPListenAddr(t *testing.T) {

	ctx, _ := contextlib.CreateNewContext(context.Background())

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)

	l, e := NewListen(ctx, "tcp4", addr)
	if e != nil {
		t.Fatalf(fmt.Sprintf(lumerinlib.FileLine()+"Listen() Test Failed: network:%s", e))
	}

	l.Run()

	netaddr, e := l.Addr()
	if e != nil {
		t.Fatalf(fmt.Sprintf(lumerinlib.FileLine()+"Addr() Test Failed: network:%s", e))
	}

	fmt.Printf(lumerinlib.FileLine()+" Addr() Returned %s:%s\n", netaddr.Network(), netaddr.String())

	if netaddr.Network() != "tcp" {
		t.Fatalf(fmt.Sprintf(lumerinlib.FileLine()+"Addr() Test Failed: network:%s", netaddr.Network()))
	}

	retaddr := netaddr.String()
	if retaddr != addr {
		t.Fatalf(fmt.Sprintf(lumerinlib.FileLine()+"Addr() Test Failed: address:%s", netaddr.String()))
	}

	l.Cancel()

}

func TestTCPSetupListenerAccept(t *testing.T) {

	ctx, _ := contextlib.CreateNewContext(context.Background())

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)

	l, e := NewListen(ctx, "tcp", addr)
	if e != nil {
		t.Fatalf("Listen() Test Failed: %s", e)
	}

	l.Run()

	fmt.Printf(lumerinlib.FileLine() + " Dialing\n")

	client, e := Dial(ctx, "tcp", addr)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Dial Test Failed: %s", e))
	}

	local, e := client.LocalAddrString()
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Dial Test Failed: %s", e))
	}
	remote, e := client.RemoteAddrString()
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Dial Test Failed: %s", e))
	}
	fmt.Printf(lumerinlib.FileLine()+" Dial completed L:%s R:%s\n", local, remote)

	server := <-l.GetAcceptChan()

	if server == nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Accept() Test Failed: %s\n", e))
	}

}

//
//
//
func TestTCPSetupListenerWrite(t *testing.T) {

	ctx, _ := contextlib.CreateNewContext(context.Background())

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)

	l, e := NewListen(ctx, "tcp", addr)
	if e != nil {
		t.Fatalf("Listen() Test Failed: %s", e)
	}

	l.Run()

	fmt.Printf(lumerinlib.FileLine() + " Dialing\n")

	client, e := Dial(ctx, "tcp", addr)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Dial Test Failed: %s", e))
	}

	local, e := client.LocalAddrString()
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Dial Test Failed: %s", e))
	}
	remote, e := client.RemoteAddrString()
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Dial Test Failed: %s", e))
	}
	fmt.Printf(lumerinlib.FileLine()+" Dial completed L:%s R:%s\n", local, remote)

	accept := <-l.GetAcceptChan()
	server := accept.(*SocketTCPStruct)
	if server == nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Accept() Test Failed: %s\n", e))
	}

	writeb := []byte(TestString)

	writeclientcount, e := client.Write(writeb)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() Test Failed: %s\n", e))
	}
	if writeclientcount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() Test Failed count: %d != %d\n", writeclientcount, len(writeb)))
	}

	writeservercount, e := server.Write(writeb)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() server Test Failed: %s\n", e))
	}
	if writeservercount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() server Test Failed count: %d != %d\n", writeservercount, len(writeb)))
	}

}

func TestTCPSetupListenerReadWrite1(t *testing.T) {
	teststring := "This is a test string"
	buflen := 1
	bufcap := 2
	test_TCPSetupListenerReadWrite(t, teststring, buflen, bufcap)
}

func TestTCPSetupListenerReadWrite2(t *testing.T) {
	teststring := "This is a test string"
	buflen := 30
	bufcap := 30
	test_TCPSetupListenerReadWrite(t, teststring, buflen, bufcap)
}

//
//
//
func test_TCPSetupListenerReadWrite(t *testing.T, teststring string, buflen int, bufcap int) {

	ctx, _ := contextlib.CreateNewContext(context.Background())

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)

	l, e := NewListen(ctx, "tcp", addr)
	if e != nil {
		t.Fatalf("Listen() Test Failed: %s", e)
	}

	l.Run()

	fmt.Printf(lumerinlib.FileLine() + " Dialing\n")

	client, e := Dial(ctx, "tcp", addr)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Dial Test Failed: %s", e))
	}

	local, e := client.LocalAddrString()
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Dial Test Failed: %s", e))
	}
	remote, e := client.RemoteAddrString()
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Dial Test Failed: %s", e))
	}
	fmt.Printf(lumerinlib.FileLine()+" Dial completed L:%s R:%s\n", local, remote)

	accept := <-l.GetAcceptChan()
	server := accept.(*SocketTCPStruct)
	if server == nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Accept() Test Failed: %s\n", e))
	}

	writeb := []byte(TestString)

	writeclientcount, e := client.Write(writeb)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() Test Failed: %s\n", e))
	}
	if writeclientcount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() Test Failed count: %d != %d\n", writeclientcount, len(writeb)))
	}

	writeservercount, e := server.Write(writeb)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() server Test Failed: %s\n", e))
	}
	if writeservercount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() server Test Failed count: %d != %d\n", writeservercount, len(writeb)))
	}

	var serverReadbuf []byte = make([]byte, buflen, bufcap)
	serverreadcount, e := server.Read(serverReadbuf)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+"Read Server Count Test Failed error:%s", e))
	}
	if serverreadcount != buflen && serverreadcount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+"Read Server Count Test Failed counts: %d, write: %d\n", serverreadcount, len(writeb)))
	}

	var clientReadbuf []byte = make([]byte, buflen, bufcap)
	clientreadcount, e := client.Read(clientReadbuf)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+"Read Client Count Test Failed error:%s", e))
	}
	if clientreadcount != buflen && clientreadcount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+"Read Client Count Test Failed counts: %d, write: %d\n", clientreadcount, len(writeb)))
	}

}

//
//
//
func TestTCPSetupListenerReadReady(t *testing.T) {

	ctx, _ := contextlib.CreateNewContext(context.Background())

	localport := getRandPort()
	addr := fmt.Sprintf("127.0.0.1:%d", localport)

	l, e := NewListen(ctx, "tcp", addr)
	if e != nil {
		t.Fatalf("Listen() Test Failed: %s", e)
	}

	l.Run()

	fmt.Printf(lumerinlib.FileLine() + " Dialing\n")

	client, e := Dial(ctx, "tcp", addr)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Dial Test Failed: %s", e))
	}

	local, e := client.LocalAddrString()
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Dial Test Failed: %s", e))
	}
	remote, e := client.RemoteAddrString()
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Dial Test Failed: %s", e))
	}
	fmt.Printf(lumerinlib.FileLine()+" Dial completed L:%s R:%s\n", local, remote)

	accept := <-l.GetAcceptChan()
	server := accept.(*SocketTCPStruct)
	if server == nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Accept() Test Failed: %s\n", e))
	}

	writeb := []byte(TestString)

	writeclientcount, e := client.Write(writeb)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() Test Failed: %s\n", e))
	}
	if writeclientcount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() Test Failed count: %d != %d\n", writeclientcount, len(writeb)))
	}

	writeservercount, e := server.Write(writeb)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() server Test Failed: %s\n", e))
	}
	if writeservercount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+" Write() server Test Failed count: %d != %d\n", writeservercount, len(writeb)))
	}

	// <-server.ReadReady()

	var serverReadbuf []byte = make([]byte, 1024)
	serverreadcount, e := server.Read(serverReadbuf)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+"Read Server Count Test Failed error:%s", e))
	}
	if serverreadcount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+"Read Server Count Test Failed counts: %d, write: %d\n", serverreadcount, len(writeb)))
	}

	// <-client.ReadReady()

	var clientReadbuf []byte = make([]byte, 1024)
	clientreadcount, e := client.Read(clientReadbuf)
	if e != nil {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+"Read Client Count Test Failed error:%s", e))
	}
	if clientreadcount != len(writeb) {
		t.Fatal(fmt.Errorf(lumerinlib.FileLine()+"Read Client Count Test Failed counts: %d, write: %d\n", clientreadcount, len(writeb)))
	}

}

//
//
//
func getRandPort() (port int) {
	port = rand.Intn(10000) + basePort
	return port
}
