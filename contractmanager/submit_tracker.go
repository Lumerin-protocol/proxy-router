package contractmanager

import (
	"time"

	"gitlab.com/TitanInd/hashrouter/data"
)

type SubmitTracker struct {
	data *data.Collection[LastSubmitModel]
}

func NewSubmitTracker() *SubmitTracker {
	return &SubmitTracker{
		data: data.NewCollection[LastSubmitModel](),
	}
}

func (t *SubmitTracker) OnSubmit(workerName string) {
	t.data.Store(LastSubmitModel{
		ID:             workerName,
		LastSubmitTime: time.Now(),
	})
}

func (t *SubmitTracker) GetLastSubmitTime(workerName string) (tm time.Time, ok bool) {
	record, ok := t.data.Load(workerName)
	if !ok {
		return time.Time{}, false
	}
	return record.LastSubmitTime, true
}

func (t *SubmitTracker) GetAll() map[string]time.Time {
	data := make(map[string]time.Time)
	t.data.Range(func(item LastSubmitModel) bool {
		data[item.ID] = item.LastSubmitTime
		return true
	})
	return data
}

type LastSubmitModel struct {
	ID             string
	LastSubmitTime time.Time
}

func (m LastSubmitModel) GetID() string {
	return m.ID
}
