package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/TitanInd/hashrouter/api"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/miner"
	"gitlab.com/TitanInd/hashrouter/tcpserver"
	"golang.org/x/sync/errgroup"
)

type App struct {
	TCPServer       *tcpserver.TCPServer
	MinerController *miner.MinerController
	Server          *api.Server
	ContractManager interfaces.ContractManager
	Logger          interfaces.ILogger
	// EventsRouter    interfaces.IEventsRouter
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

	//Bootstrap protocol layer connection handlers
	g.Go(func() error {
		a.TCPServer.SetConnectionHandler(a.MinerController)
		return a.TCPServer.Run(subCtx)
	})

	//Bootstrap contracts layer
	// g.Go(func() error {
	// 	return a.SellerManager.Run(subCtx)
	// })

	//Bootstrap API
	g.Go(func() error {
		return a.Server.Run(subCtx)
	})

	// g.Go(func() error {
	// 	return a.EventsRouter.Run()
	// })

	err := g.Wait()

	a.Logger.Warnf("App exited due to %w", err)
}
