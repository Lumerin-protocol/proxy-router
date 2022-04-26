package msgdata

import (
	"fmt"
	"testing"

	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
)

func TestAddConfigInfo(t *testing.T) {
	config := ConfigInfoJSON{
		ID:           "Test",
		DefaultDest:  "Test",
		NodeOperator: "Test",
	}

	ps := msgbus.New(10, nil)
	configRepo := NewConfigInfo(ps)
	configRepo.AddConfigInfo(config)

	if len(configRepo.ConfigInfoJSONs) != 1 {
		t.Errorf("Config Info not added")
	}
}

func TestGetAllConfigInfos(t *testing.T) {
	var config [10]ConfigInfoJSON
	for i := 0; i < 10; i++ {
		config[i].ID = "Test" + fmt.Sprint(i)
		config[i].DefaultDest = "Test"
		config[i].NodeOperator = "Test"
	}

	ps := msgbus.New(10, nil)
	configRepo := NewConfigInfo(ps)
	for i := 0; i < 10; i++ {
		configRepo.AddConfigInfo(config[i])
	}
	results := configRepo.GetAllConfigInfos()

	if len(results) != 10 {
		t.Errorf("Could not get all config infos")
	}
}

func TestGetConfigInfo(t *testing.T) {
	var config [10]ConfigInfoJSON
	for i := 0; i < 10; i++ {
		config[i].ID = "Test" + fmt.Sprint(i)
		config[i].DefaultDest = "Test"
		config[i].NodeOperator = "Test"
	}

	ps := msgbus.New(10, nil)
	configRepo := NewConfigInfo(ps)
	for i := 0; i < 10; i++ {
		configRepo.AddConfigInfo(config[i])
	}

	var results [10]ConfigInfoJSON
	var errors [10]error
	for i := 0; i < 10; i++ {
		results[i], errors[i] = configRepo.GetConfigInfo("Test" + fmt.Sprint(i))
		if errors[i] != nil {
			t.Errorf("GetConfigInfo function returned error for this ID: " + results[i].ID)
		}
	}
}

func TestUpdateConfigInfo(t *testing.T) {
	var config [10]ConfigInfoJSON
	for i := 0; i < 10; i++ {
		config[i].ID = "Test" + fmt.Sprint(i)
		config[i].DefaultDest = "Test"
		config[i].NodeOperator = "Test"
	}

	ps := msgbus.New(10, nil)
	configRepo := NewConfigInfo(ps)
	for i := 0; i < 10; i++ {
		configRepo.AddConfigInfo(config[i])
	}

	configUpdates := ConfigInfoJSON{
		ID:           "",
		DefaultDest:  "",
		NodeOperator: "Updated",
	}

	var results [10]ConfigInfoJSON
	var errors [10]error
	for i := 0; i < 10; i++ {
		errors[i] = configRepo.UpdateConfigInfo("Test"+fmt.Sprint(i), configUpdates)
		results[i], _ = configRepo.GetConfigInfo("Test" + fmt.Sprint(i))
		if errors[i] != nil {
			t.Errorf("UpdateConfigInfo function returned error for this ID: " + results[i].ID)
		}
		if results[i].NodeOperator != "Updated" {
			t.Errorf("UpdateConfigInfo function did not update Config Info")
		}
		if results[i].ID != config[i].ID {
			t.Errorf("UpdateConfigInfo function updated all Config Info instead of just filled in field")
		}
	}
}

func TestDeleteConfigInfo(t *testing.T) {
	var config [10]ConfigInfoJSON
	for i := 0; i < 10; i++ {
		config[i].ID = "Test" + fmt.Sprint(i)
		config[i].DefaultDest = "Test"
		config[i].NodeOperator = "Test"
	}

	ps := msgbus.New(10, nil)
	configRepo := NewConfigInfo(ps)
	for i := 0; i < 10; i++ {
		configRepo.AddConfigInfo(config[i])
	}

	error := configRepo.DeleteConfigInfo("Test7")
	if error != nil {
		t.Errorf("DeleteConfigInfo function returned error")
	}
	if len(configRepo.ConfigInfoJSONs) != 9 {
		t.Errorf("Config Info was not deleted")
	}
}
