package localvalidator

import (
	"testing"
)

func TestBoilerPlateFunc(t *testing.T) {
	msg, err := BoilerPlateFunc()
	if msg != "Local Validator Package" && err != nil {
		t.Fatalf("Test Failed")
	}
}
