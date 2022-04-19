package mining

import (
	"encoding/json"
	"testing"
)

func TestMiningSubscribeMethodCallPayloadUnmarshalJson(t *testing.T) {

	payload := `{"id": 56203, "method": "mining.subscribe", "params": ["bmminer/2.0.0", "5"]}`
	message := &MiningSubscribeMethodCallPayload{}

	err := json.Unmarshal([]byte(payload), message)

	if err != nil {
		t.Errorf("Failed to unmarshal json string to MiningAuthorizeMethodCallPayload: %v", err)
	}

	if message.workerNameParam != "bmminer/2.0.0" {
		t.Errorf("MiningSubscribeMethodCallPayload.workerNameParam should be equal to 'bmminer/2.0.0'; equals '%v'", message.workerNameParam)
	}

	if message.workerNumberParam != "5" {
		t.Errorf("MiningSubscribeMethodCallPayload.workerNumberParam should be equal to '5'; equals '%v'", message.workerNumberParam)
	}
}

func TestMiningAuthorizeMethodCallPayloadUnmarshalJson(t *testing.T) {

	payload := `{"id": 47, "method": "mining.authorize", "params": ["lumrrin.workername", "test"]}`
	message := &MiningAuthorizeMethodCallPayload{}

	err := json.Unmarshal([]byte(payload), message)

	if err != nil {
		t.Errorf("Failed to unmarshal json string to MiningAuthorizeMethodCallPayload: %v", err)
	}

	if message.passwordParam != "test" {
		t.Errorf("MiningAuthorizeMethodCallPayload.passwordParam should be equal to 'test'; equals '%v'", message.userParam)
	}

	if message.userParam != "lumrrin.workername" {
		t.Errorf("MiningAuthorizeMethodCallPayload.userParam should be equal to 'lumrrin.workername'; equals '%v'", message.userParam)
	}
}

func TestMiningAuthorizeMethodCallPayloadMarshalJson(t *testing.T) {

	payload := `{"id":47,"method":"mining.authorize","params":["lumrrin.workername","test"]}`
	message := &MiningAuthorizeMethodCallPayload{}

	json.Unmarshal([]byte(payload), message)

	marshaledResult, err := message.ProcessMessage("test.user", "password")
	expectedResult := `{"id": 47, "method": "mining.authorize", "params": ["test.user", "password"]}`

	if err != nil {
		t.Errorf("Failed to unmarshal MiningAuthorizeMethodCallPayload json byte array: %v", err)
	}

	resultJsonString := string(marshaledResult)

	if resultJsonString != expectedResult {
		t.Errorf("MiningAuthorizeMethodCallPayload Marshalled to JSON should be equal to '%v'; equals '%v'", expectedResult, resultJsonString)
	}
}

func TestMiningSubmitMethodCallPayloadUnmarshalJson(t *testing.T) {

	payload := `{"params": ["lumrrin.workername", "17c39b1cbf", "531f010000000000", "625e2806", "50dc5d5a"], "id": 102, "method": "mining.submit"}`
	message := &MiningSubmitMethodCallPayload{}

	err := json.Unmarshal([]byte(payload), message)

	if err != nil {
		t.Errorf("Failed to unmarshal json string to MiningSubmitMethodCallPayload: %v", err)
	}

	if message.userParam != "lumrrin.workername" {
		t.Errorf("MiningSubmitMethodCallPayload.userParam should be equal to 'lumrrin.workername'; equals '%v'", message.userParam)
	}

	if message.workIdParam != "17c39b1cbf" {
		t.Errorf("MiningSubmitMethodCallPayload.workIdParam should be equal to '6e1f'; equals '%v'", message.workIdParam)
	}
}

func TestMiningMessageUnmarshalJson(t *testing.T) {
	payload := `{"id": 47, "method": "mining.authorize", "params": ["lumrrin.workername", ""]}`
	message := &MiningMessageBase{}

	err := json.Unmarshal([]byte(payload), message)

	if err != nil {
		t.Errorf("Failed to unmarshal json string to MiningMessage: %v", err)
	}

	if message.Id != 47 {
		t.Error("MiningMessageBase.messagePayload.Id should equal 47")
	}

	if message.Method != "mining.authorize" {
		t.Error("MiningMessageBase.messagePayload.Method should equal mining.authorize")
	}

	if message.Params == nil {
		t.Error("MiningMessageBase.messagePayload.Params should not be nil")
	}

	if len(message.Params) != 2 {
		t.Error("MiningMessageBase.messagePayload.Params length should be 2")
	}

	if message.Params[0] != "lumrrin.workername" {
		t.Error("MiningMessageBase.messagePayload.Params[0] should equal 'lumrrin.workername'")
	}

	if message.Params[1] != "" {
		t.Error("MiningMessage.messagePayload.Params[1] should equal ''")
	}
}
