//go:build wireinject
// +build wireinject

package main

import (
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/wire"
	"gitlab.com/TitanInd/hashrouter/api"
	"gitlab.com/TitanInd/hashrouter/app"
	"gitlab.com/TitanInd/hashrouter/config"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/miner"
	"gitlab.com/TitanInd/hashrouter/tcpserver"
)

const VERSION = "0.01"

func main() {
	appInstance, err := InitApp()
	if err != nil {
		panic(err)
	}

	appInstance.Run()
}

func InitApp() (*app.App, error) {
	wire.Build(
		provideConfig,
		provideLogger,
		miner.NewMinerRepo,
		provideMinerController,
		provideTCPServer,
		// eventbus.NewEventBus,
		provideServer,
		// provideEthClient,
		// provideSellerContractManager,
		wire.Struct(new(app.App), "*"),
	)
	return nil, nil
}

func provideMinerController(cfg *config.Config, l interfaces.ILogger, repo *miner.MinerRepo) *miner.MinerController {
	return miner.NewMinerController(cfg.Pool.Address, cfg.Pool.User, cfg.Pool.Password, repo, l)
}

func provideTCPServer(cfg *config.Config, l interfaces.ILogger) *tcpserver.TCPServer {
	return tcpserver.NewTCPServer(cfg.Proxy.Address, l)
}

func provideServer(cfg *config.Config, l interfaces.ILogger, ph *miner.MinerController) *api.Server {
	return api.NewServer(cfg.Web.Address, l, ph)
}

func provideEthClient(cfg *config.Config) (*ethclient.Client, error) {
	return contractmanager.NewEthClient(cfg.EthNode.Address)
}

// func provideSellerContractManager(cfg *config.Config, em interfaces.IEventManager, ethClient *ethclient.Client, logger interfaces.ILogger) *contractmanager.SellerContractManager {
// 	return contractmanager.NewContractManager(logger, em, ethClient, cfg.Contract.Address, cfg.Contract.IsBuyer)
// }

func provideLogger(cfg *config.Config) (interfaces.ILogger, error) {
	return lib.NewLogger(cfg.Log.Syslog)
}

func provideConfig() (*config.Config, error) {
	var cfg config.Config
	return &cfg, config.LoadConfig(&cfg, &os.Args)
}
