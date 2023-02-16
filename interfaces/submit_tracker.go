package interfaces

import "time"

type SubmitTracker interface {
	OnSubmit(workerName string, diff int64)
	Reset(workerName string)
	GetHashRateGHS(workerName string) (hrGHS int, ok bool)
	GetLastSubmitTime(workerName string) (time time.Time, ok bool)
	GetTotalWork(workerName string) (totalWork uint64, ok bool)
	GetAll() map[string]time.Time
	Range(f func(m any) bool)
}
