package accountingmanager

import (
	"testing"
)

func TestBoilerPlateFunc(t *testing.T) {
	msg, err := BoilerPlateFunc()
	if msg != "Accounting Manager Package" && err != nil {
		t.Fatalf("Test Failed")
	}
}
