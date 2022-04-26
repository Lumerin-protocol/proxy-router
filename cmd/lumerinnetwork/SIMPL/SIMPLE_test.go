package simple

import (
	"fmt"
	"testing"
	"time"

	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/sockettcp"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
	"gitlab.com/TitanInd/lumerin/lumerinlib/testinglib"
)

//
// Test NewListen()
//
func Test_NewListen(t *testing.T) {

	l := log.New()
	mb := msgbus.New(1, l)

	ctx := testinglib.GetNewContextWithValueStruct()
	contextlib.GetContextStruct(ctx).SetLog(l)
	contextlib.GetContextStruct(ctx).SetMsgBus(mb)

	ip := "127.0.0.1"
	port := testinglib.GetRandPort()
	addr := fmt.Sprintf("%s:%d", ip, port)
	srcaddr := lumerinlib.NewNetAddr(lumerinlib.TCP, addr)
	contextlib.GetContextStruct(ctx).SetSrc(srcaddr)

	sls, e := NewListen(ctx)
	if e != nil {
		t.Errorf(lumerinlib.FileLineFunc()+" NewListen error:%s", e)
	}

	_ = sls
}

//
// Test Run() and goListenAccept()
//
func Test_Run_goListenAccept(t *testing.T) {

	l := log.New()
	mb := msgbus.New(1, l)

	ctx := testinglib.GetNewContextWithValueStruct()
	contextlib.GetContextStruct(ctx).SetLog(l)
	contextlib.GetContextStruct(ctx).SetMsgBus(mb)

	ip := "127.0.0.1"
	port := testinglib.GetRandPort()
	addr := fmt.Sprintf("%s:%d", ip, port)
	srcaddr := lumerinlib.NewNetAddr(lumerinlib.TCP, addr)
	contextlib.GetContextStruct(ctx).SetSrc(srcaddr)

	t.Logf(lumerinlib.FileLineFunc() + " NewListen()")
	sls, e := NewListen(ctx)
	if e != nil {
		t.Errorf(lumerinlib.FileLineFunc()+" NewListen error:%s", e)
	}

	t.Logf(lumerinlib.FileLineFunc() + " Listener Run()")
	sls.Run()

	//
	// Open a connection to the listener socket
	//
	t.Logf(lumerinlib.FileLineFunc() + " Dial()")
	_, e = sockettcp.Dial(ctx, "tcp", fmt.Sprintf("%s:%d", ip, port))
	if e != nil {
		t.Errorf(lumerinlib.FileLineFunc()+" sockettcp error:%s", e)
	}

	t.Logf(lumerinlib.FileLineFunc() + " Wait()")
	select {
	case <-ctx.Done():
		t.Errorf(lumerinlib.FileLineFunc() + " cacncled")
	case <-time.After(time.Second * 5):
		t.Errorf(lumerinlib.FileLineFunc() + " timeout")
	case <-sls.GetAccept():
		t.Logf(lumerinlib.FileLineFunc() + " Accepted")
	}

	t.Logf(lumerinlib.FileLineFunc() + " Done")
	sls.Close()

}

//func generateTestContext() context.Context {
//	//create a ContextStruct and add into :tab
//	ctx := context.Background()
//	cs := &lumerincontext.ContextStruct{}
//	cs.Src = generateTestAddr()
//	cs.DstID = generateTestAddr("127.0.0.1", "3333")
//	ctx = context.WithValue(ctx, lumerincontext.ContextKey, cs)
//	return ctx
//}
//
//func generateTestAddr() net.Addr {
//	conn, _ := net.Dial("tcp", "golang.org:http")
//	return conn.RemoteAddr()
//}
//
//generate a SimpleStruct for testing purposes
//func generateSimpleStruct() SimpleStruct {
//	myContext := generateTestContext()
//	mySimpleStruct := SimpleStruct{
//		ctx: myContext,
//		// cancel: dummyFunc,
//		// eventHandler:      0,
//		eventChan:  make(chan *SimpleEvent),
//		msgbusChan: make(chan *msgbus.Event),
//		// protocolChan:      make(chan *SimpleEvent),
//		// commChan:          make(chan []byte),
//		connectionMapping: make(map[ConnUniqueID]*lumerinconnection.LumerinSocketStruct),
//	}
//	return mySimpleStruct
//}

//generate a SimpleListenStruct for testing purposes
//func generateSimpleListenStruct() SimpleListenStruct {
//	myContext := generateTestContext()
//	myAddr := generateTestAddr()
//	myStruct, _ := NewListen(myContext, myAddr)
//	return myStruct
//}

/*
test steps
1. protocol layer calls the New function in the SIMPLE package
	a context and an Addr are passed into New
2. checks the error message, test fails if the error message is anything but nil

*/
// func TestInitializeSimpleListenStruct(t *testing.T) {
// 	_, err := New(generateTestContext(), generateTestAddr())
// 	if err != nil {
// 		t.Error("failed to initialize SimpleListenStruct")
// 	}
// }

/*
test steps
1. create a dummy communication layer to listen for information from a context
2. create a new simple struct routine
3. call the run function and pass in a standard context
4. check the dummy communication layer to see if the context inforation has been passed fown
*/
// func TestSimpleStructRun(t *testing.T) {
// 	simpleStruct := generateSimpleStruct()
// 	// context := generateTestContext() //this needs to be replaced to accept a context from the protocol
// 	simpleStruct.Run() //run is working but needs to do "something" with the context
// }

/*
test steps
1. create a new SimpleListenStruct
2. call the close function on the SimpleListenStruct
3. ensure all associated routines are closed
*/
//func TestSimpleListenStructClose(t *testing.T) {
//	listenStruct, _ := NewListen(generateTestContext(), generateTestAddr())
//	listenStruct.Close()
//}

/*
test steps
1. create a new SimpleListenStruct
2. call the close function on the SimpleListenStruct
3. ensure all associated routines are closed
*/
//func TestSimpleStructClose(t *testing.T) {
//	simpleStruct := generateSimpleStruct()
//	//context := generateTestContext() //this needs to be replaced to accept a context from the protocol
//	simpleStruct.Run() //run is working but needs to do "something" with the context
//	simpleStruct.Close()
//}

//func TestSetMessageSizeDefault(t *testing.T) {
//	simpleStruct := generateSimpleStruct()
//	if simpleStruct.maxMessageSize != 0 {
//		t.Errorf("message expected to be 0, actually is: %d", simpleStruct.maxMessageSize)
//	}
//	simpleStruct.SetMessageSizeDefault(100)
//	if simpleStruct.maxMessageSize != 100 {
//		t.Errorf("message expected to be 100, actually is: %d", simpleStruct.maxMessageSize)
//	}
//}

/*
testing that a SimpleStruct will dial a connection and accuratley store the resulting
connection in the mapping, and retrieve the mapping
*/
// func TestDialFunctionality(t *testing.T) {
// 	simpleStruct := generateSimpleStruct()
// 	testAddr := generateTestAddr()

//if simpleStruct.connectionIndex != 0 {
//	t.Error("testing index is not 0")
//}

//	e := simpleStruct.AsyncDial(0, testAddr)

// 	if e != nil {
// 		t.Errorf("%s", e)
// 	}

// }

/*
test to initialize a SimpleListenStruct and retrieve a SimpleStruct in the ProtocolLayer
steps:
1. create a SimpleListenStruct
2. call the Run function on the SimpleListenStruct
3. listen to the accept channel on the SimpleListenStruct
4. finish test when a SimpleStruct is detected on accept channel
*/
//func TestSimpleStructCreateOnRun(t *testing.T) {
//	simpleListenStruct := generateSimpleListenStruct()
//	go simpleListenStruct.Run()
//
//	var simpleStruct *SimpleStruct
//
//	//go routine to listen for the simpleListenStruct accept channel
//	go func() {
//		simpleStruct = <-simpleListenStruct.accept
//		t.Log("\n\n\nmeow\n\n\n")
//		t.Logf("%+v", simpleStruct)
//		if simpleStruct.eventHandler != 1 {
//			t.Error("did not create an accurate SimpleStruct")
//		}
//		//need a way to detect if the SimpleStruct was correctly generated
//	}()
//
//}

/*
test to retrieve a SimpleStruct from the SimpleListenStruct and dial a connection
steps:
1. create a SimpleListenStruct
2. run the SimpleListenStruct
3. retrieve the SimpleEvent from the SimpleListenStruct accept channel
4. call the dial function on the SimpleStruct
5. confirm that the id counter is now 1
*/
//func TestProtocolDialTheSimpleStruct(t *testing.T) {
//	simpleListenStruct := generateSimpleListenStruct()
//	simpleListenStruct.Run()
//	testAddr := generateTestAddr()
//
//	var simpleStruct *SimpleStruct
//
//	go func() {
//		simpleStruct = <-simpleListenStruct.accept
//		//initial dial
//		err := simpleStruct.AsyncDial(testAddr)
//
//		if err != nil {
//			t.Errorf("error creating a connection: %s", err)
//		}
//
//		//second dial to ensure that the uid increases as expected
//		err = simpleStruct.AsyncDial(testAddr)
//
//		if err != nil {
//			t.Errorf("error creating a connection: %s", err)
//		}
//
//	}()
//}
