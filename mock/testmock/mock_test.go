package testmock

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/mock/minermock"
	"gitlab.com/TitanInd/hashrouter/mock/poolmock"
)

func TestConnectMocks(t *testing.T) {
	workerName := "test-miner"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)

	pool := poolmock.NewPoolMock(0)

	err := pool.Connect(ctx)
	if err != nil {
		t.Log(err)
	}

	port := pool.GetPort()

	go func() {
		err := pool.Run(ctx)
		if err != nil {
			t.Log(err)
			errCh <- err
		}
	}()

	dest, _ := lib.ParseDest(fmt.Sprintf("stratum+tcp://%s:123@0.0.0.0:%d", workerName, port))

	miner := minermock.NewMinerMock(dest)
	miner.SetSubmitInterval(time.Second)

	go func() {
		err := miner.Run(ctx)
		if err != nil {
			t.Log(err)
			errCh <- err
		}
	}()

	go func() {
		err = <-errCh
		t.Error(err)
	}()

	<-time.After(5 * time.Second)

	poolConn := pool.GetConnByWorkerName(workerName)
	if poolConn == nil {
		t.Fatalf("connection not found")
	}

	submitCount := poolConn.GetSubmitCount()
	if submitCount == 0 {
		t.Fatalf("no submits sent")
	}
}
