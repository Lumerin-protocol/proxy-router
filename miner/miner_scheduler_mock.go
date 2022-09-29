package miner

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/protocol"
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

func (s *MinerSchedulerMock) Allocate(ID string, percentage float64, dest interfaces.IDestination) (*Split, error) {
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
} // get miner unique id (host:port for example)

func (s *MinerSchedulerMock) GetDestSplit() *DestSplit {
	return &s.DestSplit
}

func (s *MinerSchedulerMock) SetDestSplit(d *DestSplit) {
	s.DestSplit = *d
}

func (s *MinerSchedulerMock) GetCurrentDest() interfaces.IDestination {
	return s.Dest
}
func (s *MinerSchedulerMock) ChangeDest(dest interfaces.IDestination) error {
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

func (s *MinerSchedulerMock) GetHashRate() protocol.Hashrate {
	return nil
}

func (s *MinerSchedulerMock) GetUnallocatedHashrateGHS() int {
	return s.UnallocatedHashrateGHS
}

func (s *MinerSchedulerMock) SwitchToDefaultDestination() error {
	return nil
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

var _ MinerScheduler = new(MinerSchedulerMock)
