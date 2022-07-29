package stratumv1_message

import "encoding/json"

// Message: {"id":47,"result":true,"error":null}
// Message: {"id":1,"result":[[["mining.set_difficulty","1"],["mining.notify","1"]],"06650601bd171b",8],"error":null}
// Message: {"id":4,"result":null,"error":[-5,"Too low difficulty",null]}
type MiningResult struct {
	ID     int               `json:"id"`
	Result json.RawMessage   `json:"result"`
	Error  MiningResultError `json:"error"`
}

type MiningResultError = []json.RawMessage // data of different types: int and string

func ParseMiningResult(b []byte) (*MiningResult, error) {
	m := &MiningResult{}
	return m, json.Unmarshal(b, m)
}

func (m *MiningResult) GetID() int {
	return m.ID
}

func (m *MiningResult) SetID(ID int) {
	m.ID = ID
}

func (m *MiningResult) IsError() bool {
	return m.Error != nil
}

// Returns unparsed error field (json)
// TODO: parse error code and message correctly
func (m *MiningResult) GetError() string {
	b, _ := json.Marshal(m.Error)
	return string(b)
}

func (m *MiningResult) Serialize() []byte {
	b, _ := json.Marshal(m)
	return b
}

var _ MiningMessageGeneric = new(MiningResult)
