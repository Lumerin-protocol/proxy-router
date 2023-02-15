package interfaces

import "time"

type SubmitTracker interface {
	OnSubmit(workerName string)
	GetLastSubmitTime(workerName string) (time.Time, bool)
	GetAll() map[string]time.Time
}
