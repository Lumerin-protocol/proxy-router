package walletmanager

import (
	"testing"
)

func TestBoilerPlateFunc(t *testing.T) {
	msg, err := BoilerPlateFunc()
	if msg != "Wallet Manager Package" && err != nil {
		t.Fatalf("Test Failed")
	}
}
