package mining

// Message: {"id": 4, "method": "mining.submit", "params": ["shev8.local", "620daf25f", "0000000000000000", "62cea7a6", "f9b40000"]}
type MiningSubmit struct {
	MiningRequest
}

// type MiningSubmitParams []string TODO: parse each field

func NewMiningSubmit(msg MessageBase) (*MiningSubmit, error) {
	// params := MiningSubmitParams{}
	// err := json.Unmarshal(msg.Params, params)
	// if err != nil {
	// 	return nil, err
	// }
	miningReq, err := NewMiningRequest(msg)
	if err != nil {
		return nil, err
	}
	return &MiningSubmit{
		MiningRequest: *miningReq,
	}, nil
}

func (m *MiningSubmit) Serialize() []byte {
	return m.serialize(m)
}
