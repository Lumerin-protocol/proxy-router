package message

import "encoding/json"

// Message: {"id": 1, "method": "mining.subscribe", "params": ["cpuminer/2.5.1", "1"]}
const MethodMiningSubscribe = "mining.subscribe"

type MiningSubscribe struct {
	ID     int                    `json:"id"`
	Method string                 `json:"method,omitempty"`
	Params *miningSubscribeParams `json:"params"`
}

type miningSubscribeParams = [2]string

func NewMiningSubscribe() *MiningSubscribe {
	return &MiningSubscribe{
		Method: MethodMiningSubscribe,
		Params: &miningSubscribeParams{},
	}
}

func ParseMiningSubscribe(b []byte) (*MiningSubscribe, error) {
	m := &MiningSubscribe{}
	return m, json.Unmarshal(b, m)
}

func (m *MiningSubscribe) GetID() int {
	return m.ID
}

func (m *MiningSubscribe) SetID(ID int) {
	m.ID = ID
}

func (m *MiningSubscribe) GetWorkerName() string {
	return m.Params[0]
}

func (m *MiningSubscribe) SetWorkerName(name string) {
	m.Params[0] = name
}

func (m *MiningSubscribe) GetWorkerNumber() string {
	return m.Params[1]
}

func (m *MiningSubscribe) SetWorkerNumber(name string) {
	m.Params[1] = name
}

func (m *MiningSubscribe) Serialize() []byte {
	b, _ := json.Marshal(m)
	return b
}

var _ MiningMessageToPool = new(MiningSubscribe)
