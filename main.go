//go:build wireinject
// +build wireinject

package main

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/wire"
	"gitlab.com/TitanInd/hashrouter/api"
	"gitlab.com/TitanInd/hashrouter/app"
	"gitlab.com/TitanInd/hashrouter/config"
	"gitlab.com/TitanInd/hashrouter/connections"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/events"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/mining"
	"go.uber.org/zap"
)

const VERSION = "0.01"

func main() {
	appInstance, err := InitApp()
	if err != nil {
		panic(err)
	}

	appInstance.Run()
}

func provideMiningController(cfg *config.Config, em interfaces.IEventManager) *mining.MiningController {
	return mining.NewMiningController(cfg.Pool.User, cfg.Pool.Password, em)
}

func provideConnectionController(cfg *config.Config, mc *mining.MiningController, em interfaces.IEventManager) *connections.ConnectionsController {
	return connections.NewConnectionsController(cfg.Pool.Address, mc, em)
}

func provideServer(cfg *config.Config, cc *connections.ConnectionsController) *api.Server {
	return api.NewServer(cfg.Web.Address, cc)
}

func provideEthClient(cfg *config.Config) (*ethclient.Client, error) {
	return contractmanager.NewEthClient(cfg.EthNode.Address)
}

func provideSellerContractManager(cfg *config.Config, em interfaces.IEventManager, ethClient *ethclient.Client, logger *zap.SugaredLogger) *contractmanager.SellerContractManager {
	return contractmanager.NewSellerContractManager(logger, em, ethClient, cfg.Contract.Address)
}

func InitApp() (*app.App, error) {
	wire.Build(
		lib.NewLogger,
		config.NewConfig,
		events.NewEventManager,
		provideMiningController,
		provideConnectionController,
		provideServer,
		provideEthClient,
		provideSellerContractManager,
		wire.Struct(new(app.App), "*"),
	)
	return nil, nil
}
