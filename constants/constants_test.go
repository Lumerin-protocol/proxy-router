package constants

import "testing"

func TestConstants(t *testing.T) {
	var closeoutType CloseoutType = constantTest(11)

	t.Logf("Closeout type: %v", closeoutType)
}

func constantTest(closeoutType CloseoutType) CloseoutType {
	return 10
}
