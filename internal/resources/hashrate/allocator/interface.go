package allocator

import (
	"context"
	"net/url"
	"time"

	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate/hashrate"
	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate/proxy"
)

type StratumProxyInterface interface {
	Connect(ctx context.Context) error
	// deprecated
	ConnectDest(ctx context.Context, newDestURL *url.URL) error

	Run(ctx context.Context) error
	SetDest(ctx context.Context, dest *url.URL, onSubmit func(diff float64)) error
	SetDestWithoutAutoread(ctx context.Context, dest *url.URL, onSubmit func(diff float64)) error

	GetID() string
	GetHashrate() proxy.Hashrate
	GetDifficulty() float64
	GetDest() *url.URL
	GetSourceWorkerName() string
	GetDestWorkerName() string
	GetMinerConnectedAt() time.Time
	GetStats() map[string]int
	GetDestConns() *map[string]string
	IsVetting() bool
	VettingDone() <-chan struct{}
	GetIncomingContractID() *string
}

type HashrateFactory = func() *hashrate.Hashrate
