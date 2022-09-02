package stratumv1_message

import "encoding/json"

// Message: {"id":null,"method":"mining.set_difficulty","params":[8192]}
const MethodMiningSetExtranonce = "mining.set_extranonce"

type MiningSetExtranonce struct {
	Method string                     `json:"method,omitempty"`
	Params *miningSetExtranonceParams `json:"params"`
}

type miningSetExtranonceParams = [2]interface{}

func NewMiningSetExtranonce() *MiningSetExtranonce {
	return &MiningSetExtranonce{
		Method: MethodMiningSetExtranonce,
		Params: &miningSetExtranonceParams{},
	}
}

func ParseMiningSetExtranonce(b []byte) (*MiningSetExtranonce, error) {
	m := &MiningSetExtranonce{}
	return m, json.Unmarshal(b, m)
}

func (m *MiningSetExtranonce) GetExtranonce() (extranonce string, size int) {
	return m.Params[0].(string), m.Params[1].(int)
}

func (m *MiningSetExtranonce) SetExtranonce(extranonce string, size int) {
	m.Params[0], m.Params[1] = extranonce, size
}

func (m *MiningSetExtranonce) Serialize() []byte {
	b, _ := json.Marshal(m)
	return b
}

var _ MiningMessageGeneric = new(MiningSetExtranonce)
