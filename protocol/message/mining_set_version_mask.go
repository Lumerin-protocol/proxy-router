package message

import "encoding/json"

// Message: {"params":["00003000"], "id":null, "method": "mining.set_version_mask"}
// Only used with pools that implement https://en.bitcoin.it/wiki/BIP_0310
const MethodMiningSetVersionMask = "mining.set_version_mask"

type MiningSetVersionMask struct {
	Method string                      `json:"method,omitempty"`
	Params *miningSetVersionMaskParams `json:"params"`
}

type miningSetVersionMaskParams = [1]string

func NewMiningSetVersionMask() *MiningSetVersionMask {
	return &MiningSetVersionMask{
		Method: MethodMiningSetVersionMask,
		Params: &miningSetVersionMaskParams{},
	}
}

func ParseMiningSetVersionMask(b []byte) (*MiningSetVersionMask, error) {
	m := &MiningSetVersionMask{}
	return m, json.Unmarshal(b, m)
}

func (m *MiningSetVersionMask) GetVersionMask() string {
	return m.Params[0]
}

func (m *MiningSetVersionMask) SetVersionMask(versionMask string, size int) {
	m.Params[0] = versionMask
}

func (m *MiningSetVersionMask) Serialize() []byte {
	b, _ := json.Marshal(m)
	return b
}

var _ MiningMessageGeneric = new(MiningSetVersionMask)
