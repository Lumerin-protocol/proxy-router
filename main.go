package main

import (
	"context"
	"log"

	"gitlab.com/TitanInd/hashrouter/api"
	"gitlab.com/TitanInd/hashrouter/config"
	"gitlab.com/TitanInd/hashrouter/connections"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/events"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/mining"
)

const VERSION = "0.01"

func main() {
	ctx, _ := context.WithCancel(context.Background())

	cfg, _ := config.NewConfig()

	logger, _ := lib.NewLogger()
	defer logger.Sync()

	eventManager := events.NewEventManager()
	miningController := mining.NewMiningController(cfg.PoolUser, cfg.PoolPassword, eventManager)
	connectionsController := connections.NewConnectionsController(cfg.PoolAddress, miningController, eventManager)
	server := api.NewServer(cfg.WebAddress, connectionsController)

	ethClient, err := contractmanager.NewEthClient(cfg.EthNodeAddress)
	if err != nil {
		panic(err)
	}
	sellerManager := contractmanager.NewSellerContractManager(log.Default(), eventManager, ethClient)
	// stratum

	connectionsController.Run()
	miningController.Run()
	server.Run(ctx)
	sellerManager.Run(ctx, cfg.ContractAddress)

	<-ctx.Done()
}
