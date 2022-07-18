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
	"gitlab.com/TitanInd/hashrouter/events"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/proxyhandler"
	"gitlab.com/TitanInd/hashrouter/tcpserver"
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

func InitApp() (*app.App, error) {
	wire.Build(
		provideConfig,
		provideLogger,
		provideTCPServer,
		provideProxyHandler,
		events.NewEventManager,
		provideServer,
		provideEthClient,
		provideSellerContractManager,
		wire.Struct(new(app.App), "*"),
	)
	return nil, nil
}

func provideProxyHandler(cfg *config.Config, l *zap.SugaredLogger) *proxyhandler.ProxyHandler {
	return proxyhandler.NewProxyHandler(cfg.Pool.Address, cfg.Pool.User, cfg.Pool.Password, l)
}

func provideTCPServer(cfg *config.Config, l *zap.SugaredLogger) *tcpserver.TCPServer {
	return tcpserver.NewTCPServer(cfg.Proxy.Address, l)
}

func provideServer(cfg *config.Config, l *zap.SugaredLogger, ph *proxyhandler.ProxyHandler) *api.Server {
	return api.NewServer(cfg.Web.Address, l, ph)
}

func provideEthClient(cfg *config.Config) (*ethclient.Client, error) {
	return contractmanager.NewEthClient(cfg.EthNode.Address)
}

func provideSellerContractManager(cfg *config.Config, em interfaces.IEventManager, ethClient *ethclient.Client, logger *zap.SugaredLogger) *contractmanager.SellerContractManager {
	return contractmanager.NewSellerContractManager(logger, em, ethClient, cfg.Contract.Address)
}

func provideLogger(cfg *config.Config) (*zap.SugaredLogger, error) {
	return lib.NewLogger(cfg.Log.Syslog)
}

func provideConfig() (*config.Config, error) {
	var cfg config.Config
	return &cfg, config.LoadConfig(&cfg, &os.Args)
}
