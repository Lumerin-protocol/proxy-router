package message

import "encoding/json"

type MiningGeneric struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}

func (m *MiningGeneric) Serialize() []byte {
	bytes, _ := json.Marshal(m)
	return bytes
}

var _ MiningMessageGeneric = new(MiningGeneric)
