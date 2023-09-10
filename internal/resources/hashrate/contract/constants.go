package contract

import "errors"

var (
	ErrContractClosed = errors.New("contract closed")
)

const (
	ResourceTypeHashrate        = "hashrate"
	ResourceEstimateHashrateGHS = "hashrate_ghs"
)

type ValidationStage int8

const (
	ValidationStageNotValidating ValidationStage = 0
	ValidationStageValidating    ValidationStage = 1
	ValidationStageFinished      ValidationStage = 2
)

func (s *ValidationStage) String() string {
	switch *s {
	case ValidationStageNotValidating:
		return "not validating"
	case ValidationStageValidating:
		return "validating"
	case ValidationStageFinished:
		return "finished"
	default:
		return "unknown"
	}
}
