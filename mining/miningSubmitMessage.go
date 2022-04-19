package mining

import (
	"encoding/json"
)

type MiningSubmitMethodCallPayload struct {
	MiningMessageBase
	userParam   string
	workIdParam string
}

func (m *MiningSubmitMethodCallPayload) Process() (buf []byte, err error) {

	m.Params[0] = m.userParam

	return cleanJson(json.Marshal(m))
}

func (m *MiningSubmitMethodCallPayload) UnmarshalJSON(buf []byte) (err error) {

	miningMessageBase, err := unMarshalEmbedded(buf)

	if err != nil {
		return err
	}

	m.MiningMessageBase = *miningMessageBase

	m.userParam = m.Params[0].(string)
	m.workIdParam = m.Params[1].(string)

	return nil
}
