package walletmanager

import (
	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

func BoilerPlateFunc() (string, error) {
	msg := "Wallet Manager Package"
	return lumerinlib.BoilerPlateLibFunc(msg), nil // always returns no error
}
