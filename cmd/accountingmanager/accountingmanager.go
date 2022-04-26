package accountingmanager

import (
	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

func BoilerPlateFunc() (string, error) {
	msg := "Accounting Manager Package"
	return lumerinlib.BoilerPlateLibFunc(msg), nil // always returns no error
}
