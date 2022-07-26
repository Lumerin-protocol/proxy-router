package contractmanager

import "testing"

func TestDeployment(t *testing.T) {
	configPath := "../../ganacheconfig.json"
	mnemonic := "course surface achieve episode cable brisk flame enjoy beyond hand rival predict"
	BeforeEach(configPath, mnemonic)
}
