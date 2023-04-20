package contractmanager

import (
	"time"

	"gitlab.com/TitanInd/hashrouter/data"
	"gitlab.com/TitanInd/hashrouter/hashrate"
)

type GlobalHashrate struct {
	data *data.Collection[*WorkerHashrateModel]
}

func NewGlobalHashrate() *GlobalHashrate {
	return &GlobalHashrate{
		data: data.NewCollection[*WorkerHashrateModel](),
	}
}

func (t *GlobalHashrate) OnSubmit(workerName string, diff int64) {
	actual, _ := t.data.LoadOrStore(&WorkerHashrateModel{ID: workerName, hr: hashrate.NewHashrateV2(hashrate.NewSma(9 * time.Minute))})
	actual.OnSubmit(diff)
}

func (t *GlobalHashrate) GetLastSubmitTime(workerName string) (tm time.Time, ok bool) {
	record, ok := t.data.Load(workerName)
	if !ok {
		return time.Time{}, false
	}
	return record.hr.GetLastSubmitTime(), true
}

func (t *GlobalHashrate) GetHashRateGHS(workerName string) (hrGHS int, ok bool) {
	record, ok := t.data.Load(workerName)
	if !ok {
		return 0, false
	}
	return record.GetHashRateGHS(), true
}

func (t *GlobalHashrate) GetTotalWork(workerName string) (work uint64, ok bool) {
	record, ok := t.data.Load(workerName)
	if !ok {
		return 0, false
	}
	return record.hr.GetTotalWork(), true
}

func (t *GlobalHashrate) GetAll() map[string]time.Time {
	data := make(map[string]time.Time)
	t.data.Range(func(item *WorkerHashrateModel) bool {
		data[item.ID] = item.hr.GetLastSubmitTime()
		return true
	})
	return data
}

func (t *GlobalHashrate) Range(f func(m any) bool) {
	t.data.Range(func(item *WorkerHashrateModel) bool {
		return f(item)
	})
}

func (t *GlobalHashrate) Reset(workerName string) {
	t.data.Delete(workerName)
}

type WorkerHashrateModel struct {
	ID string
	hr *hashrate.Hashrate
}

func (m *WorkerHashrateModel) GetID() string {
	return m.ID
}

func (m *WorkerHashrateModel) OnSubmit(diff int64) {
	m.hr.OnSubmit(diff)
}

func (m *WorkerHashrateModel) GetHashRateGHS() int {
	hr, _ := m.hr.GetHashrateAvgGHSCustom(time.Duration(0))
	return hr
}

func (m *WorkerHashrateModel) GetLastSubmitTime() time.Time {
	return m.hr.GetLastSubmitTime()
}
