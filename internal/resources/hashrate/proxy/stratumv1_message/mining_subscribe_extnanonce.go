package stratumv1_message

import (
	"encoding/json"

	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate/proxy/interfaces"
)

// Message: {"id":null,"method":"mining.extranonce.subscribe","params":[]}
const MethodExtranonceSubscribe = "mining.extranonce.subscribe"

type MiningExtranonceSubscribe struct {
	ID     *int                             `json:"id"`
	Method string                           `json:"method,omitempty"`
	Params *MiningExtranonceSubscribeParams `json:"params"`
}

type MiningExtranonceSubscribeParams = []interface{}

func NewMiningExtranonceSubscribe(ID int) *MiningExtranonceSubscribe {
	return &MiningExtranonceSubscribe{
		ID:     &ID,
		Method: MethodExtranonceSubscribe,
		Params: &MiningExtranonceSubscribeParams{},
	}
}

func ParseMiningExtranonceSubscribe(b []byte) (*MiningExtranonceSubscribe, error) {
	m := &MiningExtranonceSubscribe{}
	return m, json.Unmarshal(b, m)
}

func (m *MiningExtranonceSubscribe) Serialize() []byte {
	b, _ := json.Marshal(m)
	return b
}

var _ interfaces.MiningMessageGeneric = new(MiningExtranonceSubscribe)
