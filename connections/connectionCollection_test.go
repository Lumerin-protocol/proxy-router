package connections

import (
	"testing"
)

var testMinerAddress = "testconnectionaddress"
var testPoolAddress = "testpooladdress"

func TestCreateConnection(t *testing.T) {
	col := CreateConnectionCollection()

	connection := col.CreateNewConnection(testMinerAddress, testPoolAddress, "status").(*Connection)

	if connection.Address != testMinerAddress {
		t.Errorf("Expected connection.address to equal %v; equals %v", testMinerAddress, connection.Address)
	}

	if connection.Destination != testPoolAddress {
		t.Errorf("Expected connection.destination to equal %v; equals %v", testPoolAddress, connection.Destination)
	}
}

func TestAddAndGetConnection(t *testing.T) {
	col := CreateConnectionCollection()

	err := col.AddConnection(testMinerAddress, testPoolAddress, "status")

	if err != nil {
		t.Error(err)
	}

	gotConnection := col.GetConnection(testMinerAddress).(*Connection)

	if gotConnection.Address != testMinerAddress {
		t.Errorf("Expected connection.address to equal %v; equals %v", testMinerAddress, gotConnection.Address)
	}

	if gotConnection.Destination != testPoolAddress {
		t.Errorf("Expected connection.destination to equal %v; equals %v", testPoolAddress, gotConnection.Destination)
	}
}

func TestAddAndGetConnections(t *testing.T) {
	col := CreateConnectionCollection()

	testConnectionAddress2 := testMinerAddress + "2"
	testConnectionAddress3 := testMinerAddress + "3"

	col.AddConnection(testMinerAddress, testPoolAddress, "status")
	col.AddConnection(testConnectionAddress2, testPoolAddress, "status2")
	col.AddConnection(testConnectionAddress3, testPoolAddress, "status3")

	connections, err := col.GetConnections()

	if err != nil {
		t.Errorf("failed to get connections with GetConnection function; %v", err)
	}

	gotConnection := connections[0].(*Connection)
	gotConnection2 := connections[1].(*Connection)
	gotConnection3 := connections[2].(*Connection)

	if gotConnection.Address != testMinerAddress {
		t.Errorf("Expected gotConnection.address to equal '%v'; equals '%v'", testMinerAddress, gotConnection.Address)
	}

	if gotConnection.Destination != testPoolAddress {
		t.Errorf("Expected gotConnection.destination to equal '%v'; equals '%v'", testPoolAddress, gotConnection.Destination)
	}

	if gotConnection.status != "status" {
		t.Errorf("Expected gotConnection.status to equal 'status'; equals '%v'", gotConnection.status)
	}

	if gotConnection2.Address != testConnectionAddress2 {
		t.Errorf("Expected gotConnection2.address to equal '%v'; equals '%v'", testConnectionAddress2, gotConnection2.Address)
	}

	if gotConnection2.Destination != testPoolAddress {
		t.Errorf("Expected gotConnection.destination to equal '%v'; equals '%v'", testPoolAddress, gotConnection2.Destination)
	}

	if gotConnection2.status != "status2" {
		t.Errorf("Expected gotConnection.status to equal 'status2'; equals '%v'", gotConnection2.status)
	}

	if gotConnection3.Address != testConnectionAddress3 {
		t.Errorf("Expected gotConnection3.address to equal '%v'; equals '%v'", testConnectionAddress3, gotConnection3.Address)
	}

	if gotConnection3.Destination != testPoolAddress {
		t.Errorf("Expected gotConnection3.destination to equal '%v'; equals '%v'", testPoolAddress, gotConnection3.Destination)
	}

	if gotConnection3.status != "status3" {
		t.Errorf("Expected gotConnection.status to equal 'status3'; equals '%v'", gotConnection3.status)
	}
}
