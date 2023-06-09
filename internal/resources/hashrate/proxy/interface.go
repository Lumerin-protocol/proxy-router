package proxy

import (
	"context"
	"net/url"
	"time"
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

type DestConnFactory = func(ctx context.Context, url *url.URL) (*DestConn, error)
