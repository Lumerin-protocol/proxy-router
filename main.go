//go:build wireinject
// +build wireinject

package main

import (
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"gitlab.com/TitanInd/hashrouter/api"
	"gitlab.com/TitanInd/hashrouter/app"
	"gitlab.com/TitanInd/hashrouter/config"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/contractmanager/blockchain"
	"gitlab.com/TitanInd/hashrouter/eventbus"
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

var networkSet = wire.NewSet(provideTCPServer, provideApiServer)
var protocolSet = wire.NewSet(miner.NewMinerRepo, provideMinerController, eventbus.NewEventBus)
var contractsSet = wire.NewSet(contractmanager.NewContractCollection, provideEthClient, provideEthWallet, provideEthGateway, provideSellerContractManager)

//TODO: make sure all providers initialized
func InitApp() (*app.App, error) {
	wire.Build(
		provideConfig,
		provideLogger,
		provideApiController,
		networkSet,
		protocolSet,
		contractsSet,
		wire.Struct(new(app.App), "*"),
	)
	return nil, nil
}

func provideMinerController(cfg *config.Config, l interfaces.ILogger, repo *miner.MinerRepo) (*miner.MinerController, error) {
	destination, err := lib.ParseDest(cfg.Pool.Address)
	if err != nil {
		return nil, err
	}

	return miner.NewMinerController(destination, repo, l), nil
}

func provideApiController(miners *miner.MinerRepo, contracts *contractmanager.ContractCollection) *gin.Engine {
	return api.NewApiController(miners, contracts)
}

func provideTCPServer(cfg *config.Config, l interfaces.ILogger) *tcpserver.TCPServer {
	return tcpserver.NewTCPServer(cfg.Proxy.Address, l)
}

func provideApiServer(cfg *config.Config, l interfaces.ILogger, controller *gin.Engine) *api.Server {
	return api.NewServer(cfg.Web.Address, l, controller)
}

func provideEthClient(cfg *config.Config) (*ethclient.Client, error) {
	return blockchain.NewEthClient(cfg.EthNode.Address)
}

func provideEthWallet(cfg *config.Config) (*blockchain.EthereumWallet, error) {
	return blockchain.NewEthereumWallet(cfg.Contract.Mnemonic, cfg.Contract.AccountIndex)
}

func provideEthGateway(cfg *config.Config, ethClient *ethclient.Client, ethWallet *blockchain.EthereumWallet, log interfaces.ILogger) (*blockchain.EthereumGateway, error) {
	return blockchain.NewEthereumGateway(ethClient, ethWallet.GetPrivateKey(), cfg.Contract.Address, log)
}

func provideSellerContractManager(
	cfg *config.Config,
	ethGateway *blockchain.EthereumGateway,
	ethWallet *blockchain.EthereumWallet,
	contracts *contractmanager.ContractCollection,
	log interfaces.ILogger,
) *contractmanager.ContractManager {
	return contractmanager.NewContractManager(ethGateway, log, contracts, ethWallet.GetAccountAddress(), ethWallet.GetPrivateKey())
}

func provideLogger(cfg *config.Config) (interfaces.ILogger, error) {
	return lib.NewLogger(cfg.Log.Syslog)
}

func provideConfig() (*config.Config, error) {
	var cfg config.Config
	return &cfg, config.LoadConfig(&cfg, &os.Args)
}
