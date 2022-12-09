package testctl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"gitlab.com/TitanInd/hashrouter/api"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/mock/minermock"
)

const (
	HASHROUTER_RELATIVE_PATH = "../.."
	HASHROUTER_EXEC_NAME     = "hashrouter"
)

type ProxyController struct {
	stratumPort int
	httpPort    int
	poolPort    int
	apiClient   *api.ApiClient
}

func NewProxyController(stratumPort, httpPort, poolPort int) *ProxyController {
	apiClient, _ := api.NewApiClient(fmt.Sprintf("http://0.0.0.0:%d", httpPort))
	return &ProxyController{
		stratumPort: stratumPort,
		httpPort:    httpPort,
		poolPort:    poolPort,
		apiClient:   apiClient,
	}
}

func (c *ProxyController) StartHashrouter(ctx context.Context) error {
	pwd, _ := os.Getwd()
	dir := path.Join(pwd, HASHROUTER_RELATIVE_PATH)
	exe := path.Join(dir, HASHROUTER_EXEC_NAME)

	cmd := exec.Command(exe,
		fmt.Sprintf("--proxy-address=0.0.0.0:%d", c.stratumPort),
		fmt.Sprintf("--web-address=0.0.0.0:%d", c.httpPort),
		fmt.Sprintf("--pool-address=stratum+tcp://proxy:123@0.0.0.0:%d", c.poolPort),
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

// Wait waits until the proxy is up
func (c *ProxyController) Wait(ctx context.Context) error {
	return lib.Poll(ctx, 20*time.Second, func() error {
		err := c.apiClient.Health()
		if err != nil {
			return err
		}
		log.Info("proxy is up")
		return nil
	})
}

func (c *ProxyController) CheckMinersConnected(miners map[int]*minermock.MinerMock) error {
	minersResp, err := c.apiClient.GetMiners()
	if err != nil {
		return err
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
		if mn.HashrateAvgGHS.T5m == 0 {
			return fmt.Errorf("miner(%s) not providing hashrate", m.GetWorkerName())
		}
	}

	return nil
}

func (c *ProxyController) CreateTestContract(workerName string, contractPoolPort int, hrGHS int, duration time.Duration) error {
	dest := lib.NewDest(workerName, "", "0.0.0.0", contractPoolPort)
	return c.apiClient.PostContract(dest, hrGHS, duration)
}
