package contractmanager

import (
	"context"

	snap "gitlab.com/TitanInd/hashrouter/data"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type GlobalSchedulerMock struct {
	IsDeliveringAdequateHashrateRes bool
}

func (s *GlobalSchedulerMock) Run(ctx context.Context) error {
	return nil
}

func (s *GlobalSchedulerMock) GetMinerSnapshot() *snap.AllocSnap {
	snap := snap.NewAllocSnap()
	return &snap
}

func (s *GlobalSchedulerMock) Update(contractID string, hashrateGHS int, dest interfaces.IDestination, onSubmit interfaces.IHashrate) error {
	return nil
}

func (s *GlobalSchedulerMock) IsDeliveringAdequateHashrate(ctx context.Context, targetHashrateGHS int, dest interfaces.IDestination, hashrateDiffThreshold float64) bool {
	return s.IsDeliveringAdequateHashrateRes
}
