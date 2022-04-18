package mining

import (
	"encoding/json"
	"testing"
)

func TestMiningMessageUnmarshalJson(t *testing.T) {
	payload := `{"id": 47, "method": "mining.authorize", "params": ["lumrrin.workername", ""]}`
	message := &MiningMessage{}

	err := json.Unmarshal([]byte(payload), message)

	if err != nil {
		t.Errorf("Failed to unmarshal json string to MiningMessage: %v", err)
	}

	if message.messagePayload == nil {
		t.Error("MiningMessage.messagePayload should not be nil")
	}

	if message.rawMessage != payload {
		t.Error("MiningMessage.rawMessage should equal payload")
	}

	if message.messagePayload.Id != 47 {
		t.Error("MiningMessage.messagePayload.Id should equal 47")
	}

	if message.messagePayload.Method != "mining.authorize" {
		t.Error("MiningMessage.messagePayload.Method should equal mining.authorize")
	}

	if message.messagePayload.Params == nil {
		t.Error("MiningMessage.messagePayload.Params should not be nil")
	}

	if len(message.messagePayload.Params) != 2 {
		t.Error("MiningMessage.messagePayload.Params length should be 2")
	}

	if message.messagePayload.Params[0] != "lumrrin.workername" {
		t.Error("MiningMessage.messagePayload.Params[0] should equal 'lumrrin.workername'")
	}

	if message.messagePayload.Params[1] != "" {
		t.Error("MiningMessage.messagePayload.Params[1] should equal ''")
	}
}
