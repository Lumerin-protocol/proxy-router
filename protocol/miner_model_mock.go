package protocol

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
)

type MinerModelMock struct {
	ID          string
	Dest        interfaces.IDestination
	Diff        int
	WorkerName  string
	HashrateGHS int
	ConnectedAt time.Time

	OnSubmitListenerHandle int

	RunErr        error
	ChangeDestErr error
}

func (m *MinerModelMock) Run(ctx context.Context, errCh chan error) {
	// <- m.RunErr
}
func (m *MinerModelMock) GetID() string {
	return m.ID
}
func (m *MinerModelMock) GetDest() interfaces.IDestination {
	return m.Dest
}
func (m *MinerModelMock) ChangeDest(dest interfaces.IDestination) error {
	return m.ChangeDestErr
}
func (m *MinerModelMock) GetCurrentDifficulty() int {
	return m.Diff
}
func (m *MinerModelMock) GetWorkerName() string {
	return m.WorkerName
}
func (m *MinerModelMock) GetHashRateGHS() int {
	return m.HashrateGHS
}
func (m *MinerModelMock) GetHashRate() Hashrate {
	return hashrate.NewHashrate(&lib.LoggerMock{}, time.Minute)
}
func (m *MinerModelMock) GetConnectedAt() time.Time {
	return m.ConnectedAt
}
func (m *MinerModelMock) OnSubmit(cb OnSubmitHandler) ListenerHandle {
	return ListenerHandle(m.OnSubmitListenerHandle)
}
