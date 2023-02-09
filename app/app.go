package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/TitanInd/hashrouter/api"
	"gitlab.com/TitanInd/hashrouter/config"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/miner"
	"gitlab.com/TitanInd/hashrouter/tcpserver"
	"golang.org/x/sync/errgroup"
)

type App struct {
	TCPServer       *tcpserver.TCPServer
	MinerController *miner.MinerController
	Server          *api.Server
	ContractManager *contractmanager.ContractManager
	GlobalScheduler *contractmanager.GlobalSchedulerV2
	Logger          interfaces.ILogger
	Config          *config.Config
}

func (a *App) Run(ctx context.Context) {
	a.Logger.Debugf("config: %+v\n", a.Config)
	ctx, cancel := context.WithCancel(ctx)

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

	defer func() {
		_ = a.Logger.Sync()
	}()

	g, subCtx := errgroup.WithContext(ctx)

	//Bootstrap protocol layer connection handlers
	g.Go(func() error {
		a.TCPServer.SetConnectionHandler(a.MinerController)
		return a.TCPServer.Run(subCtx)
	})

	if !a.Config.Contract.Disable {
		// Bootstrap contracts layer
		g.Go(func() error {
			err := a.ContractManager.Run(subCtx)
			a.Logger.Debugf("contract error: %v", err)
			return err
		})
	}

	// Bootstrap API
	g.Go(func() error {
		return a.Server.Run(subCtx)
	})

	g.Go(func() error {
		return a.GlobalScheduler.Run(subCtx)
	})

	a.Logger.Infof("proxyrouter is running in %s mode", getModeString(a.Config.Contract.IsBuyer))
	err := g.Wait()

	a.Logger.Warnf("App exited due to %v", err)
}

func getModeString(isBuyer bool) string {
	if isBuyer {
		return "BUYER"
	}
	return "SELLER"
}
