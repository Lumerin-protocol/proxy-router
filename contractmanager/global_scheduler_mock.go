package contractmanager

import (
	"context"

	snap "gitlab.com/TitanInd/hashrouter/data"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type GlobalSchedulerMock struct {
}

func NewGlobalSchedulerMock() *GlobalSchedulerMock {
	return &GlobalSchedulerMock{}
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
