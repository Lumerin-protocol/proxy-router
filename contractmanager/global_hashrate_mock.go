package contractmanager

import (
	"time"

	"gitlab.com/TitanInd/hashrouter/data"
)

type GlobalHashrateMock struct {
	data *data.Collection[*WorkerHashrateModelMock]

	GetHashRateGHSCallArgs [][]interface{}
}

func NewGlobalHashrateMock() *GlobalHashrateMock {
	return &GlobalHashrateMock{
		data: data.NewCollection[*WorkerHashrateModelMock](),
	}
}

func (t *GlobalHashrateMock) OnSubmit(workerName string, diff int64) {
}

func (t *GlobalHashrateMock) GetLastSubmitTime(workerName string) (tm time.Time, ok bool) {
	record, ok := t.data.Load(workerName)
	if !ok {
		return time.Time{}, false
	}
	return record.GetLastSubmitTime(), true
}

func (t *GlobalHashrateMock) GetHashRateGHS(workerName string) (hrGHS int, ok bool) {
	t.GetHashRateGHSCallArgs = append(t.GetHashRateGHSCallArgs, []interface{}{workerName})
	record, ok := t.data.Load(workerName)
	if !ok {
		return 0, false
	}
	hr, _ := record.GetHashRateGHS()
	return hr, true
}

func (t *GlobalHashrateMock) GetTotalWork(workerName string) (work uint64, ok bool) {
	record, ok := t.data.Load(workerName)
	if !ok {
		return 0, false
	}
	return record.GetTotalWork(), true
}

func (t *GlobalHashrateMock) GetAll() map[string]time.Time {
	return make(map[string]time.Time)
}

func (t *GlobalHashrateMock) LoadOrStore(item *WorkerHashrateModelMock) (actual *WorkerHashrateModelMock, loaded bool) {
	return t.data.LoadOrStore(item)
}

func (t *GlobalHashrateMock) Range(f func(m any) bool) {
	t.data.Range(func(item *WorkerHashrateModelMock) bool {
		return f(item)
	})
}

func (t *GlobalHashrateMock) Reset(workerName string) {
}

type WorkerHashrateModelMock struct {
	ID             string
	HrGHS          int
	TotalWork      uint64
	LastSubmitTime time.Time
}

func (m *WorkerHashrateModelMock) OnSubmit(diff int64) {
}
func (m *WorkerHashrateModelMock) GetID() string {
	return m.ID
}
func (m *WorkerHashrateModelMock) GetHashRateGHS() (int, bool) {
	return m.HrGHS, true
}
func (m *WorkerHashrateModelMock) GetTotalWork() uint64 {
	return m.TotalWork
}
func (m *WorkerHashrateModelMock) GetLastSubmitTime() time.Time {
	return m.LastSubmitTime
}
