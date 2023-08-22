package proxy

import (
	"context"
	"net/url"
	"time"

	i "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/interfaces"
	m "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
)

type HashrateCounter interface {
	OnSubmit(diff float64)
}

type HashrateCounterFunc func(diff float64)

type GlobalHashrateCounter interface {
	OnSubmit(workerName string, diff float64)
}

type Hashrate interface {
	GetHashrateAvgGHSCustom(ID string) (hrGHS int, ok bool)
	GetHashrateAvgGHSAll() map[string]float64
	GetTotalWork() int
	GetTotalDuration() time.Duration
	GetLastSubmitTime() time.Time
}

type DestConnFactory = func(ctx context.Context, url *url.URL, logID string) (*ConnDest, error)

type Interceptor = func(context.Context, i.MiningMessageGeneric) (i.MiningMessageGeneric, error)

type ResultHandler = func(a *m.MiningResult) (msg i.MiningMessageWithID, err error)
