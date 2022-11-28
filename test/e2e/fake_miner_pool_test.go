package fakeminerpool

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/mock/poolmock"
	"gitlab.com/TitanInd/hashrouter/test/testctl"
	"golang.org/x/sync/errgroup"
)

const (
	HASHROUTER_STRATUM_PORT = 4000
	HASHROUTER_HTTP_PORT    = 4001

	CONTRACT_WORKER_NAME = "contract"
	HASHRATE_WAIT_TIME   = 2 * time.Minute
	CONTRACT_HASHRATE    = 20000
	CONTRACT_DURATION    = 15 * time.Minute
)

func TestHashrateDelivery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l, _ := lib.NewDevelopmentLogger("info", false, false, false)
	log := l.Sugar()

	//
	// start pool
	//
	pool := poolmock.NewPoolMock(0, log.Named("pool"))
	err := pool.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	errGrp, subCtx := errgroup.WithContext(ctx)
	t.Cleanup(func() {
		cancel()
		err := errGrp.Wait()
		t.Log(err)
	})

	proxyCtl := testctl.NewProxyController(HASHROUTER_STRATUM_PORT, HASHROUTER_HTTP_PORT, pool.GetPort())
	minersCtl := testctl.NewMinersController(ctx, log)

	errGrp.Go(func() error {
		return pool.Run(subCtx)
	})
	errGrp.Go(func() error {
		return proxyCtl.StartHashrouter(subCtx)
	})

	go func() {
		err := errGrp.Wait()
		if err != nil {
			t.Log(err)
		}
	}()

	//
	// check proxy is up
	//
	err = proxyCtl.Wait(ctx)
	if err != nil {
		t.Fatal("proxy is not up")
	}

	//
	// start miners
	//
	_ = minersCtl.AddMiners(ctx, 10, HASHROUTER_STRATUM_PORT)
	errGrp.Go(func() error {
		return minersCtl.Run(subCtx)
	})

	//
	// check all miners connected
	//
	err = lib.Poll(ctx, 40*time.Second, func() error {
		return proxyCtl.CheckMinersConnected(minersCtl.GetMiners())
	})
	if err != nil {
		t.Fatal(err)
	}
	log.Info("all miners are connected")

	//
	// sleep while vetting period is in place
	//
	log.Infof("waiting %s for hashrate to go up", HASHRATE_WAIT_TIME)
	time.Sleep(HASHRATE_WAIT_TIME)

	//
	// create test contract
	//
	log.Infof("creating test contract for %d GHS for %s", CONTRACT_HASHRATE, CONTRACT_DURATION)
	err = proxyCtl.CreateTestContract(CONTRACT_WORKER_NAME, pool.GetPort(), CONTRACT_HASHRATE, CONTRACT_DURATION)
	if err != nil {
		t.Fatal(err)
	}

	// EMA_SATURATION_PERIOD time until average 5min hashrate will become accurate
	// TODO: consider using 1 min EMA to speed up the tests
	EMA_SATURATION_PERIOD := 5 * time.Minute
	log.Infof("check hashrate delivered for %s without failing", EMA_SATURATION_PERIOD)
	_ = watchHashrate(pool, t, EMA_SATURATION_PERIOD, false)

	//
	//
	durationLeft := CONTRACT_DURATION - EMA_SATURATION_PERIOD - time.Minute
	log.Infof("check hashrate delivered during the rest %s with failing", durationLeft)
	err = watchHashrate(pool, t, durationLeft, false)
	if err != nil {
		t.Fatalf("hashrate doesn't match: %s", err)
	}

	log.Info("hashrate matches expected")
	log.Infof("sleeping 2 minutes until contract ends")
	time.Sleep(2 * time.Minute)

	//
	//
	log.Infof("checking if contract ended")
	_, ok := pool.GetHRByWorkerName(CONTRACT_WORKER_NAME)
	if ok {
		t.Fatal("contract should've been already ended")
	}
}

// watchHashrate
//
// shouldFailOnWrongHR=true stops execution if hashrate is not accurate
//
// shouldFailOnWrongHR=false continues monitoring if hashrate is not accurate
func watchHashrate(pool *poolmock.PoolMock, t *testing.T, duration time.Duration, shouldFailOnWrongHR bool) error {
	var (
		i         time.Duration
		delay     = 15 * time.Second
		tolerance = 0.1
		err       error
	)

	for i = 0; i < duration; i += delay {
		time.Sleep(delay)

		hrGHS, ok := pool.GetHRByWorkerName(CONTRACT_WORKER_NAME)
		if !ok {
			err = fmt.Errorf("worker(%s) not found", CONTRACT_WORKER_NAME)
			t.Log(err)
			if shouldFailOnWrongHR {
				return err
			}
			continue
		}
		if !lib.AlmostEqual(hrGHS, CONTRACT_HASHRATE, tolerance) {
			err = fmt.Errorf("invalid hashrate expected(%d) actual(%d)", CONTRACT_HASHRATE, hrGHS)
			t.Log(err)
			if shouldFailOnWrongHR {
				return err
			}
			continue
		}
		t.Logf("hashrate (%d) is within expected range (%d ± %.1f%%)", hrGHS, CONTRACT_HASHRATE, tolerance*100)
	}

	t.Logf("monitoring finished")
	return err
}
