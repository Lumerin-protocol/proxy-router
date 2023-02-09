package protocol

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type MinerModelMock struct {
	ID          string
	Dest        interfaces.IDestination
	Diff        int
	WorkerName  string
	HashrateGHS int
	ConnectedAt time.Time

	RunErr               error
	ChangeDestErr        error
	ChangeDestCalledWith []interfaces.IDestination
}

func (m *MinerModelMock) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
func (m *MinerModelMock) GetID() string {
	return m.ID
}
func (m *MinerModelMock) GetDest() interfaces.IDestination {
	return m.Dest
}
func (m *MinerModelMock) ChangeDest(ctx context.Context, dest interfaces.IDestination, onSubmit interfaces.IHashrate) error {
	m.ChangeDestCalledWith = append(m.ChangeDestCalledWith, dest)
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
func (m *MinerModelMock) GetHashRate() interfaces.Hashrate {
	return hashrate.NewHashrate()
}
func (m *MinerModelMock) GetConnectedAt() time.Time {
	return m.ConnectedAt
}
func (m *MinerModelMock) RangeDestConn(f func(key any, value any) bool) {}
func (m *MinerModelMock) IsFaulty() bool {
	return false
}
func (m *MinerModelMock) OnFault(func(ctx context.Context)) {}
