package message

import "encoding/json"

// Message: {"id":null,"method":"mining.set_difficulty","params":[8192]}
const MethodMiningSetDifficulty = "mining.set_difficulty"

type MiningSetDifficulty struct {
	Method string                     `json:"method,omitempty"`
	Params *miningSetDifficultyParams `json:"params"`
}

type miningSetDifficultyParams = [1]int

func NewMiningSetDifficulty(difficulty int) *MiningSetDifficulty {
	return &MiningSetDifficulty{
		Method: MethodMiningSetDifficulty,
		Params: &miningSetDifficultyParams{difficulty},
	}
}

func ParseMiningSetDifficulty(b []byte) (*MiningSetDifficulty, error) {
	m := &MiningSetDifficulty{}
	return m, json.Unmarshal(b, m)
}

func (m *MiningSetDifficulty) GetDifficulty() int {
	return m.Params[0]
}

func (m *MiningSetDifficulty) SetDifficulty(difficulty int) {
	m.Params[0] = difficulty
}

func (m *MiningSetDifficulty) Serialize() []byte {
	b, _ := json.Marshal(m)
	return b
}

var _ MiningMessageGeneric = new(MiningSetDifficulty)
