package mining

import (
	"fmt"
	"math/rand"
	"testing"

	"gitlab.com/TitanInd/hashrouter/lib"
)

var (
	id         = 1
	minerId    = "test-user"
	password   = "test-pwd"
	messageRaw = []byte(fmt.Sprintf(`{"id": %d, "method": "mining.authorize", "params": ["%s", "%s"]}`, id, minerId, password))
)

func TestNewMiningAuthorize(t *testing.T) {
	// creation
	authMsg := newMiningMessage(t)

	// getters
	if authMsg.GetID() != id {
		t.Fatalf("GetID")
	}
	if authMsg.GetMinerID() != minerId {
		t.Fatalf("GetMinerID")
	}
	if authMsg.GetPassword() != password {
		t.Fatalf("GetPassword")
	}
}

func TestMinigAuthorizeSerialize(t *testing.T) {
	authMsg := newMiningMessage(t)

	serizalized := authMsg.Serialize()
	normalized, _ := lib.NormalizeJson(messageRaw)

	if string(normalized) != string(serizalized) {
		t.FailNow()
	}
}

func TestMiningAuthorizeSetters(t *testing.T) {
	authMsg := newMiningMessage(t)

	id = rand.Int()
	authMsg.SetID(id)
	if authMsg.GetID() != id {
		t.Fatalf("SetID")
	}

	minerId = "new-miner-id"
	authMsg.SetMinerID(minerId)
	if authMsg.GetMinerID() != minerId {
		t.Fatalf("SetMinerID")
	}

	password = "new-miner-id"
	authMsg.SetPassword(password)
	if authMsg.GetPassword() != password {
		t.Fatalf("SetPassword")
	}
}

func newMiningMessage(t *testing.T) *MiningAuthorize2 {
	msg, err := ParseMinerMessage(messageRaw)
	if err != nil {
		t.FailNow()
	}
	authMsg, ok := msg.(*MiningAuthorize2)
	if !ok {
		t.FailNow()
	}
	return authMsg
}

func TestMiningReq(t *testing.T) {
	var authMsg Message
	authMsg = newMiningMessage(t)
	m := authMsg.(*MiningRequest)
	id := m.GetID()
	t.Logf("ID: %d", id)
}
