package mining

import "encoding/json"

// Message: {"id":47,"result":true,"error":null}
// Message: {"id":1,"result":[[["mining.set_difficulty","1"],["mining.notify","1"]],"06650601bd171b",8],"error":null}
type MiningResult struct {
	MessageBase
	ID    int               `json:"id"`
	Error MiningResultError `json:"error"`
}

type MiningResultError = []json.RawMessage // data of different types: int and string

func NewMiningResult(m *MessageBase) (*MiningResult, error) {
	var ID int
	err := json.Unmarshal(m.ID, &ID)
	if err != nil {
		return nil, err
	}

	errorField := &MiningErrorError{}
	err = json.Unmarshal(m.Error, errorField)
	if err != nil {
		return nil, err
	}
	return &MiningResult{
		ID:          ID,
		MessageBase: *m,
		Error:       *errorField,
	}, nil
}

func (m *MiningResult) IsError() bool {
	return m.Error != nil
}

// Returns unparsed error field (json)
// TODO: parse error code and message correctly
func (m *MiningResult) GetError() string {
	return string(m.Error[1])
}

func (m *MiningResult) GetID() int {
	return m.ID
}

func (m *MiningResult) SetID(ID int) {
	m.ID = ID
}

func (m *MiningResult) Serialize() []byte {
	return m.serialize(m)
}
