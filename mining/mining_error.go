package mining

import "encoding/json"

// Message: {"id":4,"result":null,"error":[-5,"Too low difficulty",null]}
type MiningError struct {
	MessageBase
	ID int `json:"id"`
}

type MiningErrorError []json.RawMessage // data of different types: int and string

func NewMiningError(m *MessageBase) (*MiningError, error) {
	errorField := &MiningErrorError{}
	err := json.Unmarshal(m.Error, errorField)
	if err != nil {
		return nil, err
	}
	return &MiningError{MessageBase: *m}, nil
}

// Returns unparsed error field (json)
// TODO: parse error code and message correctly
func (m *MiningError) GetError() string {
	return string(m.Error)
}

func (m *MiningError) GetID() int {
	return m.ID
}

func (m *MiningError) SetID(ID int) {
	m.ID = ID
}

func (m *MiningError) Serialize() []byte {
	return m.serialize(m)
}
