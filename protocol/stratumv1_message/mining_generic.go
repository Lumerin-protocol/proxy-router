package stratumv1_message

import "encoding/json"

type MiningGeneric struct {
	ID     int             `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}

func ParseGenericMessage(b []byte) (*MiningGeneric, error) {
	m := &MiningGeneric{}
	return m, json.Unmarshal(b, m)
}

func (m *MiningGeneric) Serialize() []byte {
	bytes, _ := json.Marshal(m)
	return bytes
}

var _ MiningMessageGeneric = new(MiningGeneric)