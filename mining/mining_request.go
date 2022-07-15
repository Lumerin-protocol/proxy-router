package mining

import "encoding/json"

type MiningRequest struct {
	MessageBase
	ID     int               `json:"id"`
	Params []json.RawMessage `json:"params"`
}

func NewMiningRequest(msg MessageBase) (*MiningRequest, error) {
	var ID int
	err := json.Unmarshal(msg.ID, &ID)
	if err != nil {
		return nil, err
	}
	return &MiningRequest{
		MessageBase: msg,
		ID:          ID,
	}, nil
}

func (m *MiningRequest) GetID() int {
	return m.ID
}

func (m *MiningRequest) SetID(ID int) {
	m.ID = ID
}

func (m *MiningRequest) Serialize() []byte {
	return m.serialize(m)
}
