package stratumv1_message

import "testing"

func TestMiningUnknown(t *testing.T) {
	msg := []byte(`{"id":1,"method":"mining.configure","params":[["minimum-difficulty","version-rolling"],{"minimum-difficulty.value":2048,"version-rolling.mask":"1fffe000","version-rolling.min-bit-count":2}]}`)
	parsed, err := ParseMiningConfigure(msg)
	if err != nil {
		t.FailNow()
	}
	msg2 := parsed.Serialize()

	// quick and dirty assuming the order and formatting of fields remains the same
	// TODO: write more reliable test
	if string(msg) != string(msg2) {
		t.Fail()
	}
}
