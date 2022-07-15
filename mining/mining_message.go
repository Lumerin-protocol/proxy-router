package mining

import (
	"encoding/json"
	"fmt"
)

type Message interface {
	Serialize() []byte
}

type MinerMessage interface {
	Message
	GetID() int
	SetID(int)
}

type MessageBase struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

func ParseMinerMessage(messageRaw []byte) (MinerMessage, error) {
	message := &MessageBase{}
	err := json.Unmarshal(messageRaw, message)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal StratumV1 message: %s", string(messageRaw))
	}

	switch message.Method {
	case "mining.subscribe":
		return NewMiningSubscribe(*message)

	case "mining.authorize":
		return NewMiningAuthorize(*message)

	case "mining.submit":
		return NewMiningSubmit(*message)

	default:
		return nil, fmt.Errorf("unknown mining message: %s", string(messageRaw))
	}
}

func ParsePoolMessage(messageRaw []byte) (Message, error) {
	message := &MessageBase{}
	err := json.Unmarshal(messageRaw, message)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal StratumV1 message: %s", string(messageRaw))
	}
	if message.Method == "mining.notify" {
		return NewMiningNotify(message)
	}
	if message.Method == "mining.set_difficulty" {
		return NewMiningNotify(message)
	}
	if message.Result != nil {
		return NewMiningResult(message)
	}
	if message.Error != nil {
		return NewMiningError(message)
	}
	fmt.Printf("Unknown pool message: %s", string(messageRaw))
	return message, nil
}

func (m *MessageBase) Serialize() []byte {
	return m.serialize(m)
}

func (m *MessageBase) serialize(st interface{}) []byte {
	bytes, err := json.Marshal(st)
	if err != nil {
		// shouldn't go there
		panic(err)
	}
	return bytes
}

// authorize message payload {"id": 47, "method": "mining.authorize", "params": ["lumrrin.workername", ""]}
// submit message payload {"params": ["lumrrin.workername", "17b32a3814", "5602010000000000", "625dbc45", "e0bd5497", "00800000"], "id": 162, "method": "mining.submit"}
// mining.subscribe message payload: {"id":46,"result":[[["mining.set_difficulty","1"],["mining.notify","1"]],"2a650306f84cc6",8],"error":null}

// mining.subscribe result payload: {"id":46,"result":[[["mining.set_difficulty","1"],["mining.notify","1"]],"2a650306f84cc6",8],"error":null}
//submit/authorize pool result payload: {"id":47,"result":true,"error":null}
