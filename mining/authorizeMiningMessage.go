package mining

import (
	"encoding/json"
)

type MiningAuthorizeMethodCallPayload struct {
	MiningMessageBase
	userParam     string
	passwordParam string
}

func (m *MiningAuthorizeMethodCallPayload) Process() (buf []byte, err error) {

	m.Params[0] = m.userParam
	m.Params[1] = m.passwordParam

	return cleanJson(json.Marshal(m))
}

func (m *MiningAuthorizeMethodCallPayload) UnmarshalJSON(buf []byte) error {
	miningMessageBase, err := unMarshalEmbedded(buf)

	if err != nil {
		return err
	}

	m.MiningMessageBase = *miningMessageBase

	m.userParam = m.Params[0].(string)
	m.passwordParam = m.Params[1].(string)

	return nil
}
