package mining

import (
	"encoding/json"
	"strings"
)

type MiningMessageBase struct {
	Id     int           `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

func CreateMinerMessage(messageRaw []byte) (*MiningMessageBase, error) {
	message := &MiningMessageBase{}

	err := json.Unmarshal(messageRaw, message)

	return message, err
}

func (m *MiningMessageBase) Process() ([]byte, error) {
	return cleanJson(json.Marshal(m))
}

func (m *MiningMessageBase) ProcessMessage(user string, password string) ([]byte, error) {

	switch m.Method {

	case "mining.authorize":
		message := &MiningAuthorizeMethodCallPayload{MiningMessageBase: *m, userParam: user, passwordParam: password}
		return message.Process()

	case "mining.submit":
		message := &MiningSubmitMethodCallPayload{MiningMessageBase: *m, userParam: user}
		return message.Process()
	}

	return m.Process()
}

func unMarshalEmbedded(buf []byte) (*MiningMessageBase, error) {

	tempPayload := &MiningMessageBase{}
	err := json.Unmarshal(buf, tempPayload)

	return tempPayload, err
}

func cleanJson(buf []byte, err error) ([]byte, error) {

	if err != nil {
		return buf, err
	}

	return []byte(strings.ReplaceAll(strings.ReplaceAll(string(buf), ":", ": "), ",", ", ") + "\n"), nil
}

// authorize message payload {"id": 47, "method": "mining.authorize", "params": ["lumrrin.workername", ""]}
// submit message payload {"params": ["lumrrin.workername", "17b32a3814", "5602010000000000", "625dbc45", "e0bd5497", "00800000"], "id": 162, "method": "mining.submit"}
// mining.subscribe message payload: {"id":46,"result":[[["mining.set_difficulty","1"],["mining.notify","1"]],"2a650306f84cc6",8],"error":null}

// mining.subscribe result payload: {"id":46,"result":[[["mining.set_difficulty","1"],["mining.notify","1"]],"2a650306f84cc6",8],"error":null}
//submit/authorize pool result payload: {"id":47,"result":true,"error":null}
