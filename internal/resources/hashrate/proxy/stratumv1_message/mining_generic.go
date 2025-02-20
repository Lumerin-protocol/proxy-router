package stratumv1_message

import (
	"encoding/json"

	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate/proxy/interfaces"
)

type MiningGeneric struct {
	ID     *int            `json:"id"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
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

var _ interfaces.MiningMessageGeneric = new(MiningGeneric)
