package mining

import "encoding/json"

// Message: {"id": 2, "method": "mining.authorize", "params": ["workername", "password"]}
type MiningAuthorize2 struct {
	MiningRequest
	Params *MiningAuthorizeParams `json:"params"`
}

type MiningAuthorizeParams = [2]string

func NewMiningAuthorizeMsg(userName string, password string) *MiningAuthorize2 {
	return &MiningAuthorize2{
		MiningRequest: MiningRequest{},
		Params:        &MiningAuthorizeParams{userName, password},
	}
}

func NewMiningAuthorize(msg MessageBase) (*MiningAuthorize2, error) {
	params := &MiningAuthorizeParams{}
	err := json.Unmarshal(msg.Params, params)
	if err != nil {
		return nil, err
	}
	miningReq, err := NewMiningRequest(msg)
	if err != nil {
		return nil, err
	}
	return &MiningAuthorize2{
		MiningRequest: *miningReq,
		Params:        params,
	}, nil
}

func (m *MiningAuthorize2) GetMinerID() string {
	return m.Params[0]
}

func (m *MiningAuthorize2) SetMinerID(ID string) {
	m.Params[0] = ID
}

func (m *MiningAuthorize2) GetPassword() string {
	return m.Params[1]
}

func (m *MiningAuthorize2) SetPassword(ID string) {
	m.Params[1] = ID
}

func (m *MiningAuthorize2) Serialize() []byte {
	return m.serialize(m)
}
