package msgdata

import (
	"fmt"
	"testing"

	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
)

func TestAddContract(t *testing.T) {
	contract := ContractJSON{
		ID:                     "Test",
		State:                  "Test",
		Buyer:                  "Test",
		Price:                  100,
		Limit:                  100,
		Speed:                  100,
		Length:                 100,
		StartingBlockTimestamp: 100,
	}

	ps := msgbus.New(10, nil)
	contractRepo := NewContract(ps)
	contractRepo.AddContract(contract)

	if len(contractRepo.ContractJSONs) != 1 {
		t.Errorf("Contract struct not added")
	}
}

func TestGetAllContracts(t *testing.T) {
	var contract [10]ContractJSON
	for i := 0; i < 10; i++ {
		contract[i].ID = "Test" + fmt.Sprint(i)
		contract[i].State = "Test"
		contract[i].Buyer = "Test"
		contract[i].Price = 100
		contract[i].Limit = 100
		contract[i].Speed = 100
		contract[i].Length = 100
		contract[i].StartingBlockTimestamp = 100
	}

	ps := msgbus.New(10, nil)
	contractRepo := NewContract(ps)
	for i := 0; i < 10; i++ {
		contractRepo.AddContract(contract[i])
	}
	results := contractRepo.GetAllContracts()

	if len(results) != 10 {
		t.Errorf("Could not get all contract structs")
	}
}

func TestGetContract(t *testing.T) {
	var contract [10]ContractJSON
	for i := 0; i < 10; i++ {
		contract[i].ID = "Test" + fmt.Sprint(i)
		contract[i].State = "Test"
		contract[i].Buyer = "Test"
		contract[i].Price = 100
		contract[i].Limit = 100
		contract[i].Speed = 100
		contract[i].Length = 100
		contract[i].StartingBlockTimestamp = 100
	}

	ps := msgbus.New(10, nil)
	contractRepo := NewContract(ps)
	for i := 0; i < 10; i++ {
		contractRepo.AddContract(contract[i])
	}

	var results [10]ContractJSON
	var errors [10]error
	for i := 0; i < 10; i++ {
		results[i], errors[i] = contractRepo.GetContract("Test" + fmt.Sprint(i))
		if errors[i] != nil {
			t.Errorf("GetContract function returned error for this ID: " + results[i].ID)
		}
	}
}

func TestUpdateContract(t *testing.T) {
	var contract [10]ContractJSON
	for i := 0; i < 10; i++ {
		contract[i].ID = "Test" + fmt.Sprint(i)
		contract[i].State = "Test"
		contract[i].Buyer = "Test"
		contract[i].Price = 100
		contract[i].Limit = 100
		contract[i].Speed = 100
		contract[i].Length = 100
		contract[i].StartingBlockTimestamp = 100
	}

	ps := msgbus.New(10, nil)
	contractRepo := NewContract(ps)
	for i := 0; i < 10; i++ {
		contractRepo.AddContract(contract[i])
	}

	contractUpdates := ContractJSON{
		ID:                     "",
		State:                  "Updated",
		Buyer:                  "",
		Price:                  0,
		Limit:                  0,
		Speed:                  0,
		Length:                 0,
		StartingBlockTimestamp: 100,
	}

	var results [10]ContractJSON
	var errors [10]error
	for i := 0; i < 10; i++ {
		errors[i] = contractRepo.UpdateContract("Test"+fmt.Sprint(i), contractUpdates)
		results[i], _ = contractRepo.GetContract("Test" + fmt.Sprint(i))
		if errors[i] != nil {
			t.Errorf("UpdateContract function returned error for this ID: " + results[i].ID)
		}
		if results[i].State != "Updated" {
			t.Errorf("UpdateContract function did not update Contract Struct")
		}
		if results[i].ID != contract[i].ID {
			t.Errorf("UpdateContract function updated all Contract fields instead of just filled in field")
		}
	}
}

func TestDeleteContract(t *testing.T) {
	var contract [10]ContractJSON
	for i := 0; i < 10; i++ {
		contract[i].ID = "Test" + fmt.Sprint(i)
		contract[i].State = "Test"
		contract[i].Buyer = "Test"
		contract[i].Price = 100
		contract[i].Limit = 100
		contract[i].Speed = 100
		contract[i].Length = 100
		contract[i].StartingBlockTimestamp = 100
	}

	ps := msgbus.New(10, nil)
	contractRepo := NewContract(ps)
	for i := 0; i < 10; i++ {
		contractRepo.AddContract(contract[i])
	}

	error := contractRepo.DeleteContract("Test7")
	if error != nil {
		t.Errorf("DeleteContract function returned error")
	}
	if len(contractRepo.ContractJSONs) != 9 {
		t.Errorf("Contract was not deleted")
	}
}
