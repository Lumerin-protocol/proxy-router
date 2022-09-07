package stratumv1_message

import "encoding/json"

// Message: {"method": "mining.configure","id": 1,"params": [["minimum-difficulty", "version-rolling"],{"minimum-difficulty.value": 2048, "version-rolling.mask": "1fffe000", "version-rolling.min-bit-count": 2}]}
const MethodMiningConfigure = "mining.configure"

type MiningConfigure struct {
	ID     int                    `json:"id"`
	Method string                 `json:"method,omitempty"`
	Params *miningConfigureParams `json:"params"`
}

type miningConfigureParams = []json.RawMessage

func ParseMiningConfigure(b []byte) (*MiningConfigure, error) {
	m := &MiningConfigure{}
	return m, json.Unmarshal(b, m)
}

func (m *MiningConfigure) GetID() int {
	return m.ID
}

func (m *MiningConfigure) SetID(ID int) {
	m.ID = ID
}

func (m *MiningConfigure) Serialize() []byte {
	b, _ := json.Marshal(m)
	return b
}

var _ MiningMessageToPool = new(MiningConfigure)
