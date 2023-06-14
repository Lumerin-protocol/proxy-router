package proxy

import (
	"context"
	"io"
	"net/url"
	"time"

	i "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/interfaces"
	m "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
)

type StratumProxyInterface interface {
	Run(ctx context.Context) error
	SetDest(ctx context.Context, dest string) error

	GetID() string
	GetHashrate() float64
	GetDifficulty() float64
	GetDest() string
	GetSourceWorkerName() string
	GetDestWorkerName() string
	GetMinerConnectedAt() time.Time
}

type HashrateCounter interface {
	OnSubmit(diff float64)
}

type GlobalHashrateCounter interface {
	OnSubmit(workerName string, diff float64)
}

type DestConnFactory = func(ctx context.Context, url *url.URL) (*ConnDest, error)

type Interceptor = func(msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error)

type StratumReadWriter interface {
	Read(ctx context.Context) (i.MiningMessageGeneric, error)
	Write(ctx context.Context, msg i.MiningMessageGeneric) error
}

type StratumReadWriteCloser interface {
	io.Closer
	StratumReadWriter
}

type ResultHandler = func(a *m.MiningResult) (msg i.MiningMessageToPool, err error)
