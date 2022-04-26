package msgdata

import (
	"fmt"
	"testing"
	"time"

	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
)

func TestAddConnection(t *testing.T) {
	connection := ConnectionJSON{
		ID:        "Test",
		Miner:     "Test",
		Dest:      "Test",
		State:     "Test",
		TotalHash: 100, //"Test",
		StartDate: time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC),
	}

	ps := msgbus.New(10, nil)
	connectionRepo := NewConnection(ps)
	connectionRepo.AddConnection(connection)

	if len(connectionRepo.ConnectionJSONs) != 1 {
		t.Errorf("Connection struct not added")
	}
}

func TestGetAllConnections(t *testing.T) {
	var connection [10]ConnectionJSON
	for i := 0; i < 10; i++ {
		connection[i].ID = "Test" + fmt.Sprint(i)
		connection[i].Miner = "Test"
		connection[i].Dest = "Test"
		connection[i].State = "Test"
		connection[i].TotalHash = 100 //"Test"
		connection[i].StartDate = time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC)
	}

	ps := msgbus.New(10, nil)
	connectionRepo := NewConnection(ps)
	for i := 0; i < 10; i++ {
		connectionRepo.AddConnection(connection[i])
	}
	results := connectionRepo.GetAllConnections()

	if len(results) != 10 {
		t.Errorf("Could not get all connection structs")
	}
}

func TestGetConnection(t *testing.T) {
	var connection [10]ConnectionJSON
	for i := 0; i < 10; i++ {
		connection[i].ID = "Test" + fmt.Sprint(i)
		connection[i].Miner = "Test"
		connection[i].Dest = "Test"
		connection[i].State = "Test"
		connection[i].TotalHash = 100 //"Test"
		connection[i].StartDate = time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC)
	}

	ps := msgbus.New(10, nil)
	connectionRepo := NewConnection(ps)
	for i := 0; i < 10; i++ {
		connectionRepo.AddConnection(connection[i])
	}

	var results [10]ConnectionJSON
	var errors [10]error
	for i := 0; i < 10; i++ {
		results[i], errors[i] = connectionRepo.GetConnection("Test" + fmt.Sprint(i))
		if errors[i] != nil {
			t.Errorf("GetConnection function returned error for this ID: " + results[i].ID)
		}
	}
}

func TestUpdateConnection(t *testing.T) {
	var connection [10]ConnectionJSON
	for i := 0; i < 10; i++ {
		connection[i].ID = "Test" + fmt.Sprint(i)
		connection[i].Miner = "Test"
		connection[i].Dest = "Test"
		connection[i].State = "Test"
		connection[i].TotalHash = 100 //"Test"
		connection[i].StartDate = time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC)
	}

	ps := msgbus.New(10, nil)
	connectionRepo := NewConnection(ps)
	for i := 0; i < 10; i++ {
		connectionRepo.AddConnection(connection[i])
	}

	connectionUpdates := ConnectionJSON{
		ID:        "",
		Miner:     "Updated",
		Dest:      "",
		State:     "",
		TotalHash: 0, //"",
		StartDate: time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC),
	}

	var results [10]ConnectionJSON
	var errors [10]error
	for i := 0; i < 10; i++ {
		errors[i] = connectionRepo.UpdateConnection("Test"+fmt.Sprint(i), connectionUpdates)
		results[i], _ = connectionRepo.GetConnection("Test" + fmt.Sprint(i))
		if errors[i] != nil {
			t.Errorf("UpdateConnection function returned error for this ID: " + results[i].ID)
		}
		if results[i].Miner != "Updated" {
			t.Errorf("UpdateConnection function did not update Connection Struct")
		}
		if results[i].ID != connection[i].ID {
			t.Errorf("UpdateConnection function updated all Connection fields instead of just filled in field")
		}
	}
}

func TestDeleteConnection(t *testing.T) {
	var connection [10]ConnectionJSON
	for i := 0; i < 10; i++ {
		connection[i].ID = "Test" + fmt.Sprint(i)
		connection[i].Miner = "Test"
		connection[i].Dest = "Test"
		connection[i].State = "Test"
		connection[i].TotalHash = 0 //"Test"
		connection[i].StartDate = time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC)
	}

	ps := msgbus.New(10, nil)
	connectionRepo := NewConnection(ps)
	for i := 0; i < 10; i++ {
		connectionRepo.AddConnection(connection[i])
	}

	error := connectionRepo.DeleteConnection("Test7")
	if error != nil {
		t.Errorf("DeleteConnection function returned error")
	}
	if len(connectionRepo.ConnectionJSONs) != 9 {
		t.Errorf("Connection was not deleted")
	}
}
