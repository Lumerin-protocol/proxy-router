package stratumv1

import (
	"fmt"
	"math/rand"
	"testing"
)

//
// Provide standard JSON testing
//
//

//
//
//
func TestUnmarshalSubscribe(t *testing.T) {

	id := getRandInt()

	subscribeString := fmt.Sprintf("{\"id\":%d, \"method\":\"mining.subscribe\", \"params\":[ \"cpuminer/2.5.1\" ] }", id)

	msg := []byte(subscribeString)

	ret, err := unmarshalMsg(msg)
	if err != nil {
		t.Errorf("unmarshalMsg failed: %s", err)
	}

	switch ret.(type) {
	case *stratumRequest:
	default:
		t.Errorf("unmarshalMsg wrong type returned: %T", ret)
	}

	idResult, err := ret.(*stratumRequest).getID()
	if err != nil {
		t.Errorf("unmarshalMsg getID() failed: %s", err)
	}
	if id != idResult {
		t.Errorf("unmarshalMsg ID result mis-match: id %d, result:%d", id, idResult)

	}
}

//
//
//
func TestUnmarshalAuthorize(t *testing.T) {

	id := getRandInt()
	userid := "testrig.worker1"

	authorizeString := fmt.Sprintf("{\"id\":%d,\"method\":\"mining.authorize\",\"params\":[\"%s\",\"somePassword!\"]}", id, userid)

	msg := []byte(authorizeString)

	ret, err := unmarshalMsg(msg)
	if err != nil {
		t.Errorf("unmarshalMsg failed: %s", err)
	}

	switch ret.(type) {
	case *stratumRequest:
	default:
		t.Errorf("unmarshalMsg wrong type returned: %T", ret)
	}

	idResult, err := ret.(*stratumRequest).getID()
	if err != nil {
		t.Errorf("unmarshalMsg getID() failed: %s", err)
	}
	if id != idResult {
		t.Errorf("unmarshalMsg ID result mis-match: id %d, result:%d", id, idResult)
	}

	authResult, err := ret.(*stratumRequest).getAuthName()
	if err != nil {
		t.Errorf("unmarshalMsg getAuthName() failed: %s", err)
	}
	if userid != authResult {
		t.Errorf("unmarshalMsg ID result mis-match: auth: %s, result:%s", userid, authResult)
	}

	_, err = ret.(*stratumRequest).createRequestMsg()
	if err != nil {
		t.Errorf("createRequestMsg returne error:%s", err)
	}

}

//
//
//
func TestUnmarshalSetDifficulty(t *testing.T) {

	id := getRandInt()
	difficulty := getRandInt()

	difficultyString := fmt.Sprintf("{\"id\":%d,\"jsonrpc\":\"2.0\",\"method\":\"mining.set_difficulty\",\"params\":[%d]}", id, difficulty)

	msg := []byte(difficultyString)

	ret, err := unmarshalMsg(msg)
	if err != nil {
		t.Errorf("unmarshalMsg failed: %s", err)
	}

	switch ret.(type) {
	case *stratumRequest:
	default:
		t.Errorf("unmarshalMsg wrong type returned: %T", ret)
	}

	idResult, err := ret.(*stratumRequest).getID()
	if err != nil {
		t.Errorf("unmarshalMsg getID() failed: %s", err)
	}
	if id != idResult {
		t.Errorf("unmarshalMsg ID result mis-match: id %d, result:%d", id, idResult)
	}

	diffResult, err := ret.(*stratumRequest).getSetDifficulty()
	if err != nil {
		t.Errorf("unmarshalMsg getSetDifficulty() failed: %s", err)
	}
	if difficulty != int(diffResult) {
		t.Errorf("unmarshalMsg difficulty result mis-match: id %d, result:%d", difficulty, int(diffResult))
	}

	_, err = ret.(*stratumRequest).createRequestMsg()
	if err != nil {
		t.Errorf("createRequestMsg returne error:%s", err)
	}
}

//
//
//
func TestUnmarshalMiningNotify(t *testing.T) {

	miningId := getRandInt()

	notifyString := fmt.Sprintf("{"+ //
		"\"id\": null, "+ //
		"\"method\": \"mining.notify\", "+ //
		"\"params\": [ "+ //
		"\"%d\", "+ // JOB ID
		"\"4d16b6f85af6e2198f44ae2a6de67f78487ae5611b77c6c0440b921e00000000\", "+ // HEX-ENCODED PREV BLOCK HASH
		"\"01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff20020862062f503253482f04b8864e5008\", "+ // HEX-ENCODED PREFIX
		"\"072f736c7573682f000000000100f2052a010000001976a914d23fcdf86f7e756a64a7a9688ef9903327048ed988ac00000000\", "+ // HEX-ENCODED SUFFIX
		"[], "+ // -- MERKEL ROOT
		"\"00000002\", "+ // -- HEX-ENCODED BLOCK VERSION
		"\"1c2ac4af\", "+ // -- HEX-ENCODED NETWORK DIFFICULTY REQUIRED
		"\"504e86b9\", "+ // -- HEX-ENCODED CURRENT TIME FOR THE BLOCK
		"false "+ //--
		"]"+
		"}", miningId)

	msg := []byte(notifyString)

	ret, err := unmarshalMsg(msg)
	if err != nil {
		t.Errorf("unmarshalMsg failed: %s", err)
	}

	switch ret.(type) {
	case *stratumNotice:
	default:
		t.Errorf("unmarshalMsg wrong type returned: %T", ret)
	}

	notice := ret.(*stratumNotice)

	jobid, err := notice.getMiningNotifyJobID()
	if err != nil {
		t.Errorf("unmarshalMsg failed: %s", err)
	}

	j := fmt.Sprintf("%d", miningId)
	if jobid != j {
		t.Errorf("miningId not equal sent: %d, got: %s", miningId, j)
	}

	_, err = ret.(*stratumNotice).createNoticeMsg()
	if err != nil {
		t.Errorf("createRequestMsg returne error:%s", err)
	}

	_, err = ret.(*stratumNotice).createNoticeMiningNotify()
	if err != nil {
		t.Errorf("createRequestMsg returne error:%s", err)
	}

}

//
//
//
func TestUnmarshalMiningNotify2(t *testing.T) {

	miningId := getRandInt()

	notifyString := fmt.Sprintf("{"+ //
		"\"id\": null, "+ //
		"\"method\": \"mining.notify\", "+ //
		"\"params\": [ "+ //
		"\"%d\", "+ // JOB ID
		"\"4d16b6f85af6e2198f44ae2a6de67f78487ae5611b77c6c0440b921e00000000\", "+ // HEX-ENCODED PREV BLOCK HASH
		"\"01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff20020862062f503253482f04b8864e5008\", "+ // HEX-ENCODED PREFIX
		"\"072f736c7573682f000000000100f2052a010000001976a914d23fcdf86f7e756a64a7a9688ef9903327048ed988ac00000000\", "+ // HEX-ENCODED SUFFIX
		"["+ //
		"\"773418c442067fdd5c3caf10653537041db14d13249cab724d9d892d8427a66a\", "+ //
		"\"4126854f7bd3dc91bf666f53c35930685ee245239242ced1254f43e7b51b97e2\", "+ //
		"\"d89213f7501f4f6123c5d24403801b7d978957e9ecbee82869fefb295025caff\", "+ //
		"\"b4817f2f1e86914186c5acf715db97f753b84b9cc2cbd3a977e021df09ccf46d\", "+ //
		"\"51c91bbfb65e328063dbfe020913a5e92c2973796f7cd84c74806e33eaf48116\", "+ //
		"\"6e006d18ed55017612adf0e334b94d52e16b06f11adb14058a91caee161a304f\", "+ //
		"\"633c5a641b57c0fc0fc9ed669d04686634f17ff34b6d509cc9a50c58e7cd9771\", "+ //
		"\"e90773f4f44dc4a6a13e60956cad1612549e5c23a8f4ba42e760eb8661177464\", "+ //
		"\"de5fc02be1faa3dbbb59e9799ea1fae886ab25e6b154413d2e2d35204fedbaf2\", "+ //
		"\"79b109bdf26dd068446afa66c62f7d5ba30b179fcf032bb299f5a2591e0e3fce\", "+ //
		"\"fe16f0630558f6564ec212ed700b1d5469b0a9d1cd39f4b7ce344d3d01d650b7\", "+ //
		"\"03802c6be8643a09f8f74254ebf6f3704cfc622ab55f94687299fc32ca4a31da\" "+ //
		"], "+ // -- MERKEL ROOT
		"\"00000002\", "+ // -- HEX-ENCODED BLOCK VERSION
		"\"1c2ac4af\", "+ // -- HEX-ENCODED NETWORK DIFFICULTY REQUIRED
		"\"504e86b9\", "+ // -- HEX-ENCODED CURRENT TIME FOR THE BLOCK
		"false "+ //--
		"]"+
		"}", miningId)

	msg := []byte(notifyString)

	ret, err := unmarshalMsg(msg)
	if err != nil {
		t.Errorf("unmarshalMsg failed: %s", err)
	}

	switch ret.(type) {
	case *stratumNotice:
	default:
		t.Errorf("unmarshalMsg wrong type returned: %T", ret)
	}

	notice := ret.(*stratumNotice)

	jobid, err := notice.getMiningNotifyJobID()
	if err != nil {
		t.Errorf("unmarshalMsg failed: %s", err)
	}

	j := fmt.Sprintf("%d", miningId)
	if jobid != j {
		t.Errorf("miningId not equal sent: %d, got: %s", miningId, j)
	}

	_, err = ret.(*stratumNotice).createNoticeMsg()
	if err != nil {
		t.Errorf("createRequestMsg returne error:%s", err)
	}

	_, err = ret.(*stratumNotice).createNoticeMiningNotify()
	if err != nil {
		t.Errorf("createRequestMsg returne error:%s", err)
	}

}

// ---------------------------------------------------------------------------------------------

//
// getRandInt()
//
func getRandInt() (id int) {
	id = rand.Intn(1000000) + 1
	return
}
