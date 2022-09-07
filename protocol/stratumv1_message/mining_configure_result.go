package stratumv1_message

import (
	"encoding/json"
)

// {"id":151769,"jsonrpc":"2.0","result":{ "version-rolling": true, "version-rolling.mask": "1fffe000" },"error":null}

type MiningConfigureResult struct {
	ID     int                         `json:"id"`
	Result miningConfigureResultResult `json:"result"`
	Error  miningConfigureResultError  `json:"error"`
}

type miningConfigureResultResult = struct {
	VersionRolling     bool   `json:"version-rolling"`
	VersionRollingMask string `json:"version-rolling.mask"`
}
type miningConfigureResultError = interface{} // null

func ParseMiningConfigureResult(b []byte) (*MiningConfigureResult, error) {
	m := &MiningConfigureResult{}
	if err := json.Unmarshal(b, m); err != nil {
		return nil, err
	}
	return m, nil
}

func ToMiningConfigureResult(m *MiningResult) (*MiningConfigureResult, error) {
	result := &miningConfigureResultResult{}
	err := json.Unmarshal(m.Result, result)
	if err != nil {
		return nil, err
	}
	return &MiningConfigureResult{
		ID:     m.ID,
		Result: *result,
		Error:  m.Error,
	}, nil
}

func (m *MiningConfigureResult) GetID() int {
	return m.ID
}

func (m *MiningConfigureResult) SetID(ID int) {
	m.ID = ID
}

func (m *MiningConfigureResult) IsError() bool {
	return false
}

// Returns unparsed error field (json)
// TODO: parse error code and message correctly
func (m *MiningConfigureResult) GetError() string {
	return ""
}

func (m *MiningConfigureResult) GetVersionRolling() bool {
	return m.Result.VersionRolling
}

func (m *MiningConfigureResult) GetVersionRollingMask() string {
	return m.Result.VersionRollingMask
}

func (m *MiningConfigureResult) Serialize() []byte {
	b, _ := json.Marshal(m)
	return b
}

var _ MiningMessageGeneric = new(MiningConfigureResult)
