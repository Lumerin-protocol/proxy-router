package testmock

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/mock/minermock"
	"gitlab.com/TitanInd/hashrouter/mock/poolmock"
	"golang.org/x/sync/errgroup"
)

func TestConnectMocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l, _ := lib.NewDevelopmentLogger("debug", false, false, false)
	log := l.Sugar()

	pool := poolmock.NewPoolMock(0, lib.NewTestLogger())
	err := pool.Connect(ctx)
	if err != nil {
		t.Log(err)
	}

	MinerNumber := 20
	miners := make(map[string]*minermock.MinerMock, MinerNumber)

	for i := 0; i < MinerNumber; i++ {
		workerName := fmt.Sprintf("mock-miner-%d", i)
		dest := lib.MustParseDest(fmt.Sprintf(
			"stratum+tcp://%s:123@0.0.0.0:%d", workerName, pool.GetPort(),
		))
		miner := minermock.NewMinerMock(dest, log.Named(workerName))
		miner.SetSubmitInterval(5 * time.Second)
		miners[workerName] = miner
	}

	errGrp, subCtx := errgroup.WithContext(ctx)

	errGrp.Go(func() error {
		err := pool.Run(subCtx)
		if err != nil {
			t.Log(err)
		}
		return err
	})

	for _, miner := range miners {
		m := miner
		errGrp.Go(func() error {
			err := m.Run(subCtx)
			if err != nil {
				t.Log(err)
			}
			return err
		})
	}

	<-time.After(60 * time.Second)

	for workerName := range miners {
		poolConn := pool.GetConnByWorkerName(workerName)
		if poolConn == nil {
			t.Errorf("miner %s not found", workerName)
			continue
		}

		submitCount := poolConn.GetSubmitCount()
		if submitCount == 0 {
			t.Errorf("no submits sent for miner %s", workerName)
		}
	}
}
