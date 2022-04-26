package msgbus

import (
	"fmt"
	"regexp"
	"testing"
	"time"
)

func TestBoilerPlateFunc(t *testing.T) {
	eventChan := make(EventChan)
	go func(eventChan EventChan) {
		for event := range eventChan {
			fmt.Printf("Read Chan: %+v\n", event)
		}

		fmt.Printf("Closed Read Chan\n")

	}(eventChan)
	defer close(eventChan)

	config := ConfigInfo{
		ID:           "ConfigID01",
		DefaultDest:  "DestID01",
		NodeOperator: "NOID01",
	}
	dest := Dest{
		ID:     DestID(DEFAULT_DEST_ID),
		NetUrl: DestNetUrl("stratum+tcp://127.0.0.1:3334/"),
	}
	nodeOp := NodeOperator{
		ID:                     "SellerID01",
		DefaultDest:            "DestID01",
		TotalAvailableHashRate: 0,
		UnusedHashRate:         0,
		Contracts:              make(map[ContractID]ContractState),
	}
	contract := Contract{}
	miner := Miner{}
	connection := Connection{}

	ps := New(1, nil)
	pubSetParams := []struct {
		msg  MsgType
		id   IDString
		data interface{}
	}{
		{ConfigMsg, "configMsg01", config},
		{DestMsg, "destMsg01", dest},
		{NodeOperatorMsg, "sellerMsg01", nodeOp},
		{ContractMsg, "contractMsg01", contract},
		{MinerMsg, "minerMsg01", miner},
		{ConnectionMsg, "connectionMsg01", connection},
	}

	for _, params := range pubSetParams {
		if _, err := ps.Pub(params.msg, params.id, params.data); err != nil {
			t.Errorf("trying to pub: %v", err)
		}
	}

	subParams := []struct {
		msg MsgType
		id  IDString
		ch  EventChan
	}{
		{ConfigMsg, "configMsg01", eventChan},
		{DestMsg, "destMsg01", eventChan},
		{NodeOperatorMsg, "sellerMsg01", eventChan},
		{ContractMsg, "contractMsg01", eventChan},
		{MinerMsg, "minerMsg01", eventChan},
		{ConnectionMsg, "connectionMsg01", eventChan},
	}

	for _, params := range subParams {
		if _, err := ps.Sub(params.msg, params.id, params.ch); err != nil {
			t.Errorf("trying to sub: %v", err)
		}
	}

	for _, params := range pubSetParams {
		if _, err := ps.Set(params.msg, params.id, params.data); err != nil {
			t.Errorf("trying to set: %v", err)
		}
	}

	getParams := []struct {
		msg MsgType
		id  IDString
		ch  EventChan
	}{
		{ConfigMsg, "", eventChan},
		{DestMsg, "", eventChan},
		{NodeOperatorMsg, "", eventChan},
		{ContractMsg, "", eventChan},
		{MinerMsg, "", eventChan},
		{ConnectionMsg, "", eventChan},
		{ConfigMsg, "configMsg01", eventChan},
		{DestMsg, "destMsg01", eventChan},
		{NodeOperatorMsg, "sellerMsg01", eventChan},
		{ContractMsg, "contractMsg01", eventChan},
		{MinerMsg, "minerMsg01", eventChan},
		{ConnectionMsg, "connectionMsg01", eventChan},
	}

	for _, params := range getParams {
		if _, err := ps.Get(params.msg, params.id, params.ch); err != nil {
			t.Errorf("trying to get: %v", err)
		}
	}

	if _, err := ps.Shutdown(); err != nil {
		t.Errorf("shutting down: %v", err)
	}

	// symbolizes that the app runs after the message bus shuts down
	time.Sleep(time.Second * 5)
}

func TestRequestID(t *testing.T) {
	mb := New(1, nil)

	compareID := func(method string, id, requestID int) {
		if id != requestID {
			t.Errorf("expected ID %d but got %d in %s method\n", id, requestID, method)
		}
	}

	for id := 1; id < 100; id++ {
		requestID, _ := mb.Pub(NoMsg, IDString("0"), "datadatadata")
		compareID("Pub", id, requestID)
	}

	eventChan := make(EventChan)
	for id := 100; id < 200; id++ {
		requestID, _ := mb.Sub(NoMsg, IDString("0"), eventChan)
		compareID("Sub", id, requestID)
	}

	eventChan = make(EventChan)
	for id := 200; id < 300; id++ {
		requestID, _ := mb.Get(NoMsg, IDString("0"), eventChan)
		compareID("Get", id, requestID)
	}

	eventChan = make(EventChan)
	for id := 300; id < 400; id++ {
		requestID, _ := mb.SearchIP(NoMsg, "ip address", eventChan)
		compareID("SearchIP", id, requestID)
	}

	eventChan = make(EventChan)
	for id := 400; id < 500; id++ {
		requestID, _ := mb.SearchMAC(NoMsg, "mac address", eventChan)
		compareID("SearchMAC", id, requestID)
	}

	eventChan = make(EventChan)
	for id := 500; id < 600; id++ {
		requestID, _ := mb.SearchName(NoMsg, "name", eventChan)
		compareID("SearchName", id, requestID)
	}

	for id := 600; id < 700; id++ {
		requestID, _ := mb.Set(NoMsg, IDString("0"), "datadatadata")
		compareID("Set", id, requestID)
	}

	for id := 700; id < 800; id++ {
		requestID, _ := mb.Unpub(NoMsg, IDString("0"))
		compareID("Unpub", id, requestID)
	}

	eventChan = make(EventChan)
	for id := 800; id < 900; id++ {
		requestID, _ := mb.Unsub(NoMsg, IDString("0"), eventChan)
		compareID("Unsub", id, requestID)
	}

	eventChan = make(EventChan)
	requestID, _ := mb.RemoveAndCloseEventChan(eventChan)
	compareID("RemoveAndCloseEventChan", 900, requestID)

	mb = New(1, nil)
	requestID, _ = mb.Shutdown()
	compareID("Shutdown", 1, requestID)
}

func TestGetRandomIDString(t *testing.T) {
	requiredRegex := `^[0-9a-fA-F]{8}\-[0-9a-fA-F]{8}\-[0-9a-fA-F]{8}\-[0-9a-fA-F]{8}$`
	regex, err := regexp.Compile(requiredRegex)
	if err != nil {
		t.Errorf("compiling regex: %v", err)
	}

	// run 100 tests
	for i := 0; i < 100; i++ {
		testID := GetRandomIDString()

		if matched := regex.Match([]byte(testID)); !matched {
			t.Errorf("GetRandomIDString returned an incorrectly formatted string: %v", testID)
		}
	}
}
