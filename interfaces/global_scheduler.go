package interfaces

import (
	"context"

	"gitlab.com/TitanInd/hashrouter/data"
)

type IGlobalScheduler interface {
	Run(ctx context.Context) error
	GetMinerSnapshot() *data.AllocSnap
	Update(contractID string, hashrateGHS int, dest IDestination, onSubmit IHashrate) error
	IsDeliveringAdequateHashrate(ctx context.Context, targetHashrateGHS int, dest IDestination, hashrateDiffThreshold float64) bool
}
