package mining

import "encoding/json"

// Message: {"id":null,"method":"mining.set_difficulty","params":[8192]}
type MiningSetDifficulty struct {
	MessageBase
	Params MiningSetDifficultyParams `json:"params"`
}

type MiningSetDifficultyParams = [1]int

func NewMiningSetDifficulty(msg MessageBase) (*MiningSetDifficulty, error) {
	params := &MiningSetDifficultyParams{}
	err := json.Unmarshal(msg.Params, params)
	if err != nil {
		return nil, err
	}

	return &MiningSetDifficulty{
		MessageBase: msg,
		Params:      *params,
	}, nil
}

func (m *MiningSetDifficulty) GetDifficulty() int {
	return m.Params[0]
}

func (m *MiningSetDifficulty) SetDifficulty(difficulty int) {
	m.Params[0] = difficulty
}

func (m *MiningSetDifficulty) Serialize() []byte {
	return m.serialize(m)
}
