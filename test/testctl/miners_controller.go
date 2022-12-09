package testctl

import (
	"context"
	"fmt"
	"sync/atomic"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/mock/minermock"
	"golang.org/x/sync/errgroup"
)

type MinersController struct {
	miners  map[int]*minermock.MinerMock
	counter atomic.Uint32
	log     interfaces.ILogger
	errGrp  *errgroup.Group
	grpCtx  context.Context
}

func NewMinersController(ctx context.Context, log interfaces.ILogger) *MinersController {
	errGrp, grpCtx := errgroup.WithContext(ctx)

	return &MinersController{
		miners:  make(map[int]*minermock.MinerMock),
		counter: atomic.Uint32{},
		errGrp:  errGrp,
		grpCtx:  grpCtx,
		log:     log,
	}
}

// Run runs all miners and exits when all done. Should be called after AddMiners
func (c *MinersController) Run(ctx context.Context) error {
	return c.errGrp.Wait()
}

func (c *MinersController) AddMiners(ctx context.Context, count int, destPort int, hrGHS int, hrError float64) error {
	for i := 0; i < count; i++ {
		minerId := c.counter.Add(1)
		workerName := fmt.Sprintf("mock-miner-%d", minerId)
		dest := lib.MustParseDest(fmt.Sprintf(
			"stratum+tcp://%s:123@0.0.0.0:%d", workerName, destPort,
		))
		miner := minermock.NewMinerMock(dest, c.log.Named(workerName))
		miner.SetMinerGHS(hrGHS)
		miner.SetMinerError(hrError)
		c.miners[i] = miner
	}

	for _, miner := range c.miners {
		m := miner
		c.errGrp.Go(func() error {
			return m.Run(c.grpCtx)
		})
	}

	return nil
}

func (c *MinersController) GetMiners() map[int]*minermock.MinerMock {
	return c.miners
}
