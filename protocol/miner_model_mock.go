package protocol

import (
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type MinerModelMock struct {
	ID          string
	Dest        interfaces.IDestination
	Diff        int
	WorkerName  string
	HashrateGHS int

	OnSubmitListenerHandle int

	RunErr        error
	ChangeDestErr error
}

func (m *MinerModelMock) Run() error {
	return nil
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
func (m *MinerModelMock) OnSubmit(cb OnSubmitHandler) ListenerHandle {
	return ListenerHandle(m.OnSubmitListenerHandle)
}
