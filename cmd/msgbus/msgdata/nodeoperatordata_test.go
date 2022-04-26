package msgdata

import (
	"fmt"
	"testing"

	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
)

func TestAddNodeOperator(t *testing.T) {
	nodeOperator := NodeOperatorJSON{
		ID:                     "Test",
		DefaultDest:            "Test",
		TotalAvailableHashRate: 100,
		UnusedHashRate:         100,
	}
	nodeOperator.Contracts = map[msgbus.ContractID]msgbus.ContractState{
		"0x50937C047DB93CB5C87F65B6EFFEA47D03DF0F7D": msgbus.ContRunningState,
		"0xFB610E4C269DA110C97B92F5F34EAA50E5F3D500": msgbus.ContAvailableState,
		"0x397729E80F77BA09D930FE24E8D1FC74372E86D3": msgbus.ContAvailableState,
	}

	ps := msgbus.New(10, nil)
	nodeOperatorRepo := NewNodeOperator(ps)
	nodeOperatorRepo.AddNodeOperator(nodeOperator)

	if len(nodeOperatorRepo.NodeOperatorJSONs) != 1 {
		t.Errorf("NodeOperator struct not added")
	}
}

func TestGetAllNodeOperators(t *testing.T) {
	var nodeOperator [10]NodeOperatorJSON
	for i := 0; i < 10; i++ {
		nodeOperator[i].ID = "Test" + fmt.Sprint(i)
		nodeOperator[i].DefaultDest = "Test"
		nodeOperator[i].TotalAvailableHashRate = 100
		nodeOperator[i].UnusedHashRate = 100
		nodeOperator[i].Contracts = map[msgbus.ContractID]msgbus.ContractState{
			"0x50937C047DB93CB5C87F65B6EFFEA47D03DF0F7D": msgbus.ContRunningState,
			"0xFB610E4C269DA110C97B92F5F34EAA50E5F3D500": msgbus.ContAvailableState,
			"0x397729E80F77BA09D930FE24E8D1FC74372E86D3": msgbus.ContAvailableState,
		}
	}

	ps := msgbus.New(10, nil)
	nodeOperatorRepo := NewNodeOperator(ps)
	for i := 0; i < 10; i++ {
		nodeOperatorRepo.AddNodeOperator(nodeOperator[i])
	}
	results := nodeOperatorRepo.GetAllNodeOperators()

	if len(results) != 10 {
		t.Errorf("Could not get all nodeOperator structs")
	}
}

func TestGetNodeOperator(t *testing.T) {
	var nodeOperator [10]NodeOperatorJSON
	for i := 0; i < 10; i++ {
		nodeOperator[i].ID = "Test" + fmt.Sprint(i)
		nodeOperator[i].DefaultDest = "Test"
		nodeOperator[i].TotalAvailableHashRate = 100
		nodeOperator[i].UnusedHashRate = 100
		nodeOperator[i].Contracts = map[msgbus.ContractID]msgbus.ContractState{
			"0x50937C047DB93CB5C87F65B6EFFEA47D03DF0F7D": msgbus.ContRunningState,
			"0xFB610E4C269DA110C97B92F5F34EAA50E5F3D500": msgbus.ContAvailableState,
			"0x397729E80F77BA09D930FE24E8D1FC74372E86D3": msgbus.ContAvailableState,
		}
	}

	ps := msgbus.New(10, nil)
	nodeOperatorRepo := NewNodeOperator(ps)
	for i := 0; i < 10; i++ {
		nodeOperatorRepo.AddNodeOperator(nodeOperator[i])
	}

	var results [10]NodeOperatorJSON
	var errors [10]error
	for i := 0; i < 10; i++ {
		results[i], errors[i] = nodeOperatorRepo.GetNodeOperator("Test" + fmt.Sprint(i))
		if errors[i] != nil {
			t.Errorf("GetNodeOperator function returned error for this ID: " + results[i].ID)
		}
	}
}

func TestUpdateNodeOperator(t *testing.T) {
	var nodeOperator [10]NodeOperatorJSON
	for i := 0; i < 10; i++ {
		nodeOperator[i].ID = "Test" + fmt.Sprint(i)
		nodeOperator[i].DefaultDest = "Test"
		nodeOperator[i].TotalAvailableHashRate = 100
		nodeOperator[i].UnusedHashRate = 100
		nodeOperator[i].Contracts = map[msgbus.ContractID]msgbus.ContractState{
			"0x50937C047DB93CB5C87F65B6EFFEA47D03DF0F7D": msgbus.ContRunningState,
			"0xFB610E4C269DA110C97B92F5F34EAA50E5F3D500": msgbus.ContAvailableState,
			"0x397729E80F77BA09D930FE24E8D1FC74372E86D3": msgbus.ContAvailableState,
		}
	}

	ps := msgbus.New(10, nil)
	nodeOperatorRepo := NewNodeOperator(ps)
	for i := 0; i < 10; i++ {
		nodeOperatorRepo.AddNodeOperator(nodeOperator[i])
	}

	nodeOperatorUpdates := NodeOperatorJSON{
		ID:                     "",
		DefaultDest:            "",
		TotalAvailableHashRate: 10001,
		UnusedHashRate:         0,
	}
	nodeOperatorUpdates.Contracts = map[msgbus.ContractID]msgbus.ContractState{}

	var results [10]NodeOperatorJSON
	var errors [10]error
	for i := 0; i < 10; i++ {
		errors[i] = nodeOperatorRepo.UpdateNodeOperator("Test"+fmt.Sprint(i), nodeOperatorUpdates)
		results[i], _ = nodeOperatorRepo.GetNodeOperator("Test" + fmt.Sprint(i))
		if errors[i] != nil {
			t.Errorf("UpdateNodeOperator function returned error for this ID: " + results[i].ID)
		}
		if results[i].TotalAvailableHashRate != 10001 {
			t.Errorf("UpdateNodeOperator function did not update NodeOperator Struct")
		}
		if results[i].ID != nodeOperator[i].ID {
			t.Errorf("UpdateNodeOperator function updated all NodeOperator fields instead of just filled in field")
		}
	}
}

func TestDeleteNodeOperator(t *testing.T) {
	var nodeOperator [10]NodeOperatorJSON
	for i := 0; i < 10; i++ {
		nodeOperator[i].ID = "Test" + fmt.Sprint(i)
		nodeOperator[i].DefaultDest = "Test"
		nodeOperator[i].TotalAvailableHashRate = 100
		nodeOperator[i].UnusedHashRate = 100
		nodeOperator[i].Contracts = map[msgbus.ContractID]msgbus.ContractState{
			"0x50937C047DB93CB5C87F65B6EFFEA47D03DF0F7D": msgbus.ContRunningState,
			"0xFB610E4C269DA110C97B92F5F34EAA50E5F3D500": msgbus.ContAvailableState,
			"0x397729E80F77BA09D930FE24E8D1FC74372E86D3": msgbus.ContAvailableState,
		}
	}

	ps := msgbus.New(10, nil)
	nodeOperatorRepo := NewNodeOperator(ps)
	for i := 0; i < 10; i++ {
		nodeOperatorRepo.AddNodeOperator(nodeOperator[i])
	}

	error := nodeOperatorRepo.DeleteNodeOperator("Test7")
	if error != nil {
		t.Errorf("DeleteNodeOperator function returned error")
	}
	if len(nodeOperatorRepo.NodeOperatorJSONs) != 9 {
		t.Errorf("NodeOperator was not deleted")
	}
}
