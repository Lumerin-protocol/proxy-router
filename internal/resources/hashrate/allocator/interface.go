package allocator

import (
	"context"
	"net/url"
	"time"
)

type StratumProxyInterface interface {
	Connect(ctx context.Context) error
	Run(ctx context.Context) error
	SetDest(ctx context.Context, dest *url.URL, onSubmit func(diff float64)) error

	GetID() string
	GetHashrate() float64
	GetDifficulty() float64
	GetDest() *url.URL
	GetSourceWorkerName() string
	GetDestWorkerName() string
	GetMinerConnectedAt() time.Time
	GetStats() interface{}
	GetDestConns() *map[string]string
}
