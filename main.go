//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/wire"
	"gitlab.com/TitanInd/hashrouter/api"
	"gitlab.com/TitanInd/hashrouter/app"
	"gitlab.com/TitanInd/hashrouter/config"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/contractmanager/blockchain"
	"gitlab.com/TitanInd/hashrouter/data"
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

var dataSet = wire.NewSet(data.NewTransactionsChannel, data.NewInMemoryDataStore)
var networkSet = wire.NewSet(provideTCPServer, provideServer)
var protocolSet = wire.NewSet(miner.NewMinerRepo, provideMinerController, eventbus.NewEventBus, provideConnectionsService)
var contractsSet = wire.NewSet(provideContractsRepository, blockchain.NewBlockchainWallet, provideEthClient, blockchain.NewBlockchainGateway, provideContractFactory, provideContractsGateway, contractmanager.NewNodeOperator, contractmanager.NewContractsService, provideSellerContractManager)
var hashrateCalculationSet = wire.NewSet(provideHashrateCalculator)

//TODO: make sure all providers initialized
func InitApp() (*app.App, error) {
	wire.Build(
		provideConfig,
		provideLogger,
		dataSet,
		networkSet,
		protocolSet,
		hashrateCalculationSet,
		contractsSet,
		wire.Struct(new(app.App), "*"),
	)
	return nil, nil
}

func provideContractsRepository(logger interfaces.ILogger, dataStore data.Store, transactionsChannel data.TransactionsChannel) contractmanager.IContractsRepository {
	return data.NewInMemoryRepository[interfaces.ISellerContractModel](logger, dataStore, transactionsChannel)
}

func provideContractsGateway(repo contractmanager.IContractsRepository) interfaces.IContractsGateway {
	return contractmanager.NewContractsGateway(repo)
}

func provideConnectionsService() interfaces.IConnectionsService {
	return nil
}

func provideHashrateCalculator() interfaces.IValidatorsService {

	return nil
}

func provideMinerController(cfg *config.Config, l interfaces.ILogger, repo *miner.MinerRepo) (*miner.MinerController, error) {

	destination, err := lib.ParseDest(cfg.Pool.Address)

	if err != nil {
		return nil, err
	}

	return miner.NewMinerController(destination, repo, l), nil
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

func provideSellerContractManager(
	contractsService interfaces.IContractsService,
	cfg *config.Config,
	em interfaces.IEventManager,
	factory interfaces.IContractFactory,
	ethClient *ethclient.Client,
	logger interfaces.ILogger,
	nodeOperator *contractmanager.NodeOperator,
	wallet interfaces.IBlockchainWallet,
) (interfaces.ContractManager, error) {
	return contractmanager.NewContractManager(context.TODO(), contractsService, logger, cfg, em, factory, ethClient, nodeOperator, wallet) //cfg.Contract.Address,  cfg.Contract.IsBuyer
}

func provideLogger(cfg *config.Config) (interfaces.ILogger, error) {
	return lib.NewLogger(cfg.Log.Syslog)
}

func provideConfig() (*config.Config, error) {
	var cfg config.Config
	return &cfg, config.LoadConfig(&cfg, &os.Args)
}

func provideContractFactory() interfaces.IContractFactory {
	return &ContractFactory{}
}

type ContractFactory struct {
}

func (*ContractFactory) CreateContract(
	IsSeller bool,
	ID string,
	State string,
	Buyer string,
	Price int,
	Limit int,
	Speed int,
	Length int,
	StartingBlockTimestamp int,
	Dest string,
) (interfaces.ISellerContractModel, error) {
	model, err := initContractModel()

	if err != nil {
		return model, err
	}

	model.IsSeller = IsSeller
	model.ID = ID
	model.State = State
	model.State = State
	model.Buyer = Buyer
	model.Price = Price
	model.Limit = Limit
	model.Speed = Speed
	model.Length = Length
	model.StartingBlockTimestamp = StartingBlockTimestamp

	dest, err := lib.ParseDest(Dest)

	model.Dest = dest

	// CurrentNonce

	return model, err
}

func initContractModel() (*contractmanager.Contract, error) {
	wire.Build(
		provideLogger,
		provideConfig,
		dataSet,
		protocolSet,
		contractsSet,
		provideContractModel,
	)
	return nil, nil
}

func provideContractModel(logger interfaces.ILogger, ethereumGateway interfaces.IBlockchainGateway, contractsGateway interfaces.IContractsGateway, miningService *miner.MinerController) *contractmanager.Contract {
	return &contractmanager.Contract{
		Logger:                logger,
		EthereumGateway:       ethereumGateway,
		ContractsGateway:      contractsGateway,
		RoutableStreamService: miningService,
	}
}
