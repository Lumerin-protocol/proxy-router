package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/TitanInd/hashrouter/api"
	"gitlab.com/TitanInd/hashrouter/connections"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type App struct {
	ConnectionsController *connections.ConnectionsController
	Server                *api.Server
	SellerManager         *contractmanager.SellerContractManager
	Logger                *zap.SugaredLogger
}

func (a *App) Run() {
	ctx, cancel := context.WithCancel(context.Background())

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-shutdownChan
		a.Logger.Infof("Received signal: %s", s)
		cancel()
	}()

	defer a.Logger.Sync()

	g, subCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return a.ConnectionsController.Run(subCtx)
	})

	g.Go(func() error {
		return a.SellerManager.Run(subCtx)
	})

	g.Go(func() error {
		return a.Server.Run(subCtx)
	})

	err := g.Wait()

	a.Logger.Warnf("App exited due to ", err)
}
