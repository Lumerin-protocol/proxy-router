package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/TitanInd/hashrouter/api"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/miner"
	"gitlab.com/TitanInd/hashrouter/tcpserver"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type App struct {
	TCPServer       *tcpserver.TCPServer
	MinerController *miner.MinerController
	Server          *api.Server
	SellerManager   *contractmanager.SellerContractManager
	Logger          *zap.SugaredLogger
}

func (a *App) Run() {
	ctx, cancel := context.WithCancel(context.Background())

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-shutdownChan
		a.Logger.Infof("Received signal: %s", s)
		cancel()

		s = <-shutdownChan
		a.Logger.Infof("Received signal: %s. Forcing exit...", s)
		os.Exit(1)
	}()

	defer a.Logger.Sync()

	g, subCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		a.TCPServer.SetConnectionHandler(a.MinerController)
		return a.TCPServer.Run(subCtx)
	})

	// g.Go(func() error {
	// 	return a.SellerManager.Run(subCtx)
	// })

	g.Go(func() error {
		return a.Server.Run(subCtx)
	})

	err := g.Wait()

	a.Logger.Warnf("App exited due to %w", err)
}
