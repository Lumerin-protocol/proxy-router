package miner

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
)

type MinerSchedulerMock struct {
	ID                     string
	DestSplit              DestSplit
	Dest                   lib.Dest
	Diff                   int
	WorkerName             string
	HashrateGHS            int
	UnallocatedHashrateGHS int
	ConnectedAt            time.Time
}

func NewMinerSchedulerMock() MinerSchedulerMock {
	return MinerSchedulerMock{}
}

func (s *MinerSchedulerMock) Allocate(ID string, percentage float64, dest interfaces.IDestination) (*SplitItem, error) {
	return nil, nil
}

func (s *MinerSchedulerMock) Deallocate(ID string) (ok bool) {
	return true
}

func (s *MinerSchedulerMock) Run(context.Context) error {
	return nil
}

func (s *MinerSchedulerMock) GetID() string {
	return s.ID
}

func (s *MinerSchedulerMock) GetCurrentDestSplit() *DestSplit {
	return &s.DestSplit
}

func (s *MinerSchedulerMock) GetDestSplit() *DestSplit {
	return &s.DestSplit
}

func (s *MinerSchedulerMock) GetUpcomingDestSplit() *DestSplit {
	return nil
}

func (S *MinerSchedulerMock) SetDestSplit(*DestSplit) {}

func (s *MinerSchedulerMock) GetCurrentDest() interfaces.IDestination {
	return s.Dest
}

func (s *MinerSchedulerMock) ChangeDest(ctx context.Context, dest interfaces.IDestination, contractID string, onSubmit interfaces.IHashrate) error {
	return nil
}

func (s *MinerSchedulerMock) GetCurrentDifficulty() int {
	return s.Diff

}
func (s *MinerSchedulerMock) GetWorkerName() string {
	return s.WorkerName
}

func (s *MinerSchedulerMock) GetHashRateGHS() int {
	return s.HashrateGHS
}

func (s *MinerSchedulerMock) GetHashRate() interfaces.Hashrate {
	return nil
}

func (s *MinerSchedulerMock) GetUnallocatedHashrateGHS() int {
	return s.UnallocatedHashrateGHS
}

func (s *MinerSchedulerMock) GetConnectedAt() time.Time {
	return s.ConnectedAt
}

func (s *MinerSchedulerMock) GetStatus() MinerStatus {
	return MinerStatusFree
}

func (s *MinerSchedulerMock) IsVetted() bool {
	return true
}

func (s *MinerSchedulerMock) GetUptime() time.Duration {
	return time.Hour
}

func (s *MinerSchedulerMock) RangeDestConn(f func(a, b any) bool) {}

func (s *MinerSchedulerMock) RangeHistory(f func(a HistoryItem) bool) {}

func (s *MinerSchedulerMock) RangeHistoryContractID(ID string, f func(a HistoryItem) bool) {}

var _ MinerScheduler = new(MinerSchedulerMock)
