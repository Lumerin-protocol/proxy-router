package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"syscall"
	"testing"
	"time"

	"gitlab.com/TitanInd/hashrouter/api"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/mock/minermock"
	"gitlab.com/TitanInd/hashrouter/mock/poolmock"
	"golang.org/x/sync/errgroup"
)

const (
	HASHROUTER_RELATIVE_PATH = "../.."
	HASHROUTER_EXEC_NAME     = "hashrouter"
	HASHROUTER_STRATUM_PORT  = 4000
	HASHROUTER_HTTP_PORT     = 4001

	CONTRACT_WORKER_NAME = "contract"
	HASHRATE_WAIT_TIME   = 2 * time.Minute
	CONTRACT_HASHRATE    = 20000
)

func TestHashrateDelivery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l, _ := lib.NewDevelopmentLogger("info", false, false, false)
	log := l.Sugar()

	//
	// start default pool
	//
	defaultPool := poolmock.NewPoolMock(0, log.Named("default-pool"))
	err := defaultPool.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	//
	// start contract pool
	//
	contractPool := poolmock.NewPoolMock(0, log.Named("contract-pool"))
	err = contractPool.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	errGrp, subCtx := errgroup.WithContext(ctx)
	t.Cleanup(func() {
		cancel()
		err := errGrp.Wait()
		t.Log(err)
	})

	errGrp.Go(func() error {
		return defaultPool.Run(subCtx)
	})
	errGrp.Go(func() error {
		return contractPool.Run(subCtx)
	})
	errGrp.Go(func() error {
		return StartHashrouter(subCtx, HASHROUTER_STRATUM_PORT, HASHROUTER_HTTP_PORT, defaultPool.GetPort())
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
	client, err := api.NewApiClient(fmt.Sprintf("http://0.0.0.0:%d", HASHROUTER_HTTP_PORT))
	if err != nil {
		t.Fatal(err)
	}
	err = poll(ctx, 20*time.Second, func() error {
		err := client.Health()
		if err != nil {
			return err
		}
		log.Info("proxy is up")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	//
	// start miners
	//
	MinerNumber := 10
	miners := make(map[int]*minermock.MinerMock, MinerNumber)

	for i := 0; i < MinerNumber; i++ {
		workerName := fmt.Sprintf("mock-miner-%d", i)
		dest := lib.MustParseDest(fmt.Sprintf(
			"stratum+tcp://%s:123@0.0.0.0:%d", workerName, HASHROUTER_STRATUM_PORT,
		))
		miner := minermock.NewMinerMock(dest, log.Named(workerName))
		miner.SetSubmitInterval(10 * time.Second)
		miners[i] = miner
	}

	for _, miner := range miners {
		m := miner
		errGrp.Go(func() error {
			return m.Run(subCtx)
		})
	}

	//
	// check all miners connected
	//
	err = poll(ctx, 40*time.Second, func() error {
		minersResp, err := client.GetMiners()
		if err != nil {
			t.Fatal(err)
		}

		connectedMiners := make(map[string]*api.Miner)
		for _, minerItem := range minersResp.Miners {
			connectedMiners[minerItem.WorkerName] = &minerItem
		}

		for _, m := range miners {
			mn, isOk := connectedMiners[m.GetWorkerName()]
			if !isOk {
				return fmt.Errorf("miner(%s) not connected", m.GetWorkerName())
			}
			if mn.TotalHashrateGHS == 0 {
				return fmt.Errorf("miner(%s) not providing hashrate", m.GetWorkerName())
			}
		}
		log.Info("all miners are connected")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	//
	// sleep while vetting period is in place
	//
	log.Infof("waiting %s for hashrate to go up", HASHRATE_WAIT_TIME)
	time.Sleep(HASHRATE_WAIT_TIME)

	//
	// create test contract
	//
	dest := lib.NewDest(CONTRACT_WORKER_NAME, "", "0.0.0.0", contractPool.GetPort())
	err = client.PostContract(dest, CONTRACT_HASHRATE, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	//
	// check hashrate delivered
	//
	err = poll(ctx, 30*time.Second, func() error {
		conn := contractPool.GetConnByWorkerName(CONTRACT_WORKER_NAME)
		if conn == nil {
			return fmt.Errorf("contract worker not found")
		}
		hrGHS := conn.GetHashRate()
		if !lib.AlmostEqual(hrGHS, CONTRACT_HASHRATE, 0.1) {
			return fmt.Errorf("invalid hashrate expected(%d) actual(%d)", CONTRACT_HASHRATE, hrGHS)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	log.Info("hashrate matches expected")
}

func StartHashrouter(ctx context.Context, stratumPort int, httpPort int, poolPort int) error {
	pwd, _ := os.Getwd()
	dir := path.Join(pwd, HASHROUTER_RELATIVE_PATH)
	exe := path.Join(dir, HASHROUTER_EXEC_NAME)

	cmd := exec.Command(exe,
		fmt.Sprintf("--proxy-address=0.0.0.0:%d", stratumPort),
		fmt.Sprintf("--web-address=0.0.0.0:%d", httpPort),
		fmt.Sprintf("--pool-address=stratum+tcp://proxy:123@0.0.0.0:%d", poolPort),
		fmt.Sprintf("--miner-vetting-duration=%s", time.Minute),
		"--contract-disable=true",
		"--log-color=false",
		"--log-to-file=false",
		"--log-level=info",
	)

	cmd.Dir = dir

	var b bytes.Buffer

	cmd.Stdout = io.MultiWriter(os.Stdout, &b)
	cmd.Stderr = cmd.Stdout

	err := cmd.Start()
	if err != nil {
		return err
	}

	errCh := make(chan error)

	go func() {
		errCh <- cmd.Wait()
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		err = cmd.Process.Signal(syscall.SIGINT)
		if err != nil {
			return err
		}
		return <-errCh
	}
}

func poll(ctx context.Context, dur time.Duration, f func() error) error {
	pollInterval := time.Second
	for i := 0; ; i++ {
		err := f()
		if err == nil {
			return nil
		}

		elapsed := time.Duration(i) * pollInterval
		if err != nil && elapsed > dur {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
