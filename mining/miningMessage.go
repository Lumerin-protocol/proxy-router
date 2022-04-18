package mining

import (
	"encoding/json"
	"fmt"
)

type MiningMessage struct {
	rawMessage     string
	messagePayload *MiningMethodCallPayload
}

func (m *MiningMessage) UnmarshalJSON(buf []byte) (err error) {

	m.rawMessage = string(buf)
	m.messagePayload, err = unMarshalEmbedded(buf)

	if err != nil {
		return fmt.Errorf("MiningMessage.UnmarshalJSON Failed to unmarshal MiningMessage.messagePayload from byte array - Inner Exception: %w", err)
	}

	return nil
}

type TMiningMethodCallPayload interface {
	MiningSubscribeMethodCallPayload | MiningAuthorizeMethodCallPayload
}

type miningMethodCallPayload struct {
	Id     int      `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type MiningMethodCallPayload struct {
	miningMethodCallPayload
}

func unMarshalEmbedded(buf []byte) (*MiningMethodCallPayload, error) {
	tempPayload := &MiningMethodCallPayload{}
	err := json.Unmarshal(buf, &tempPayload.miningMethodCallPayload)

	return tempPayload, err
}

type MiningSubscribeMethodCallPayload struct {
	// MiningMethodCallPayload
	workerNameParam   string
	workerNumberParam int
}

// func (m *MiningSubscribeMethodCallPayload) UnmarshalJSON(buf []byte) (err error) {
// 	return jsonToMiningMethodCallPayload(buf, &m.MiningMethodCallPayload, []interface{}{&m.workerNameParam, &m.workerNumberParam})
// }

type MiningAuthorizeMethodCallPayload struct {
	// MiningMethodCallPayload
	userParam     string
	passwordParam string
}

// func (m *MiningAuthorizeMethodCallPayload) UnmarshalJSON(buf []byte) (err error) {
// 	return jsonToMiningMethodCallPayload(buf, &m.MiningMethodCallPayload, []interface{}{&m.userParam, &m.passwordParam})
// }

// type MiningSubmitMethodCallPayload struct {
// 	MiningMethodCallPayload
// 	userParam   string
// 	workIdParam string
// }

// func (m *MiningSubmitMethodCallPayload) UnmarshalJSON(buf []byte) (err error) {
// 	return jsonToMiningMethodCallPayload(buf, &m.MiningMethodCallPayload, []interface{}{&m.userParam, &m.workIdParam})
// }

func jsonToMiningMethodCallPayload(buf []byte, instance *MiningMethodCallPayload, params interface{}) (err error) {
	instance, err = unMarshalEmbedded(buf)

	if err != nil {
		return err
	}

	if err := json.Unmarshal(buf, params); err != nil {
		return err
	}

	return nil
}

// authorize message payload {"id": 47, "method": "mining.authorize", "params": ["lumrrin.workername", ""]}
// submit message payload {"params": ["lumrrin.workername", "17b32a3814", "5602010000000000", "625dbc45", "e0bd5497", "00800000"], "id": 162, "method": "mining.submit"}
// mining.subscribe message payload: {"id":46,"result":[[["mining.set_difficulty","1"],["mining.notify","1"]],"2a650306f84cc6",8],"error":null}

// mining.subscribe result payload: {"id":46,"result":[[["mining.set_difficulty","1"],["mining.notify","1"]],"2a650306f84cc6",8],"error":null}
//submit/authorize pool result payload: {"id":47,"result":true,"error":null}
