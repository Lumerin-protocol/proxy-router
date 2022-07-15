package mining

import "encoding/json"

// Message: {"id": 1, "method": "mining.subscribe", "params": ["cpuminer/2.5.1", "1"]}
type MiningSubscribe2 struct {
	MiningRequest
	Params *MiningSubscribeParams `json:"params"`
}

type MiningSubscribeParams [2]string //TODO: figure out meaning of the fields and provide getters/setters

func NewMiningSubscribe(msg MessageBase) (*MiningSubscribe2, error) {
	params := &MiningSubscribeParams{}
	err := json.Unmarshal(msg.Params, params)
	if err != nil {
		return nil, err
	}
	return &MiningSubscribe2{
		MiningRequest: MiningRequest{MessageBase: msg},
		Params:        params,
	}, nil
}

func (m *MiningSubscribe2) GetWorkerName() string {
	return m.Params[0]
}

func (m *MiningSubscribe2) SetWorkerName(name string) {
	m.Params[0] = name
}

func (m *MiningSubscribe2) GetWorkerNumber() string {
	return m.Params[1]
}

func (m *MiningSubscribe2) SetWorkerNumber(name string) {
	m.Params[1] = name
}

func (m *MiningSubscribe2) Serialize() []byte {
	return m.serialize(m)
}
