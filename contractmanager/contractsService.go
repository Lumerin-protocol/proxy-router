package contractmanager

import (
	"gitlab.com/TitanInd/hashrouter/config"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type ContractsService struct {
	logger             interfaces.ILogger
	validatorService   interfaces.IValidatorsService
	connectionsService interfaces.IConnectionsService
	blockchainGateway  interfaces.IBlockchainGateway
	factory            interfaces.IContractFactory
	contractGateway    interfaces.IContractsGateway

	configuration *config.Config
	handlers      []func(contract interfaces.IContractModel)
}

func (service *ContractsService) ContractExists(id string) bool {
	contract, err := service.contractGateway.GetContract(id)

	if err != nil {
		return false
	}

	return contract != nil
}

func (service *ContractsService) CheckHashRate(contractId string) bool {
	// check for miners delivering hashrate for this contract
	totalHashrate := uint64(0)
	contract, err := service.GetContract(contractId)
	if err != nil {
		service.logger.Errorf("Failed to get all miners, %v", err)
	}

	// contract := contractResult

	totalHashrate, err = service.validatorService.GetHashrate()
	// TODO: create source/contract relationship
	// miners, err := buyer.Ps.GetMiners()

	// if err != nil {
	// 	//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to get all miners, Fileline::%s, Error::", lumerinlib.FileLine()), err)
	// }

	// for _, miner := range miners {
	// 	if err != nil {
	// 		//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to get miner, Fileline::%s, Error::", lumerinlib.FileLine()), err)
	// 	}
	// 	if _, ok := miner.Contracts[contractId]; ok {
	// 		totalHashrate += int(float64(miner.CurrentHashRate) * miner.Contracts[contractId])
	// 	}
	// }

	// hashrateTolerance := float64(HASHRATE_LIMIT) / 100
	// promisedHashrateMin := int(float64(contract.Speed) * (1 - hashrateTolerance))

	//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate being sent to contract %s: %d\n", contractId, totalHashrate)
	if totalHashrate <= contract.GetPromisedHashrateMin() { // promisedHashrateMin {
		//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Closing out contract %s for not meeting hashrate requirements\n", contractId)

		err := service.blockchainGateway.SetContractCloseOut(contract)

		if err != nil {
			//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Contract Close Out failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
		}

		return true
	}

	return false
}

func (service *ContractsService) GetContract(contractId string) (interfaces.IContractModel, error) {
	return service.contractGateway.GetContract(contractId)
}

func (service *ContractsService) CreateDestination(destinationUrl string) {
	panic("ContractsService.CreateDestination not implemented")
}

func (service *ContractsService) GetDestinations() []string {
	panic("ContractsService.Getdestinations not implemented")
}

func (service *ContractsService) GetHashrate() uint64 {
	panic("ContractsService.GetHashrate not implemented")
}

func (service *ContractsService) SaveContracts(models []interfaces.IContractModel) ([]interfaces.IContractModel, error) {
	for i, contract := range models {
		contract, err := contract.Save()

		if err != nil {
			return models, err
		}

		models[i] = contract
	}

	return models, nil
}

var _ interfaces.IContractsService = (*ContractsService)(nil)

// Event Listeners

func (service *ContractsService) OnContractCreated(handler func(newContract interfaces.IContractModel)) {
	service.handlers = append(service.handlers, handler)
}

//	Event handlers
func (service *ContractsService) HandleContractClosed(contract interfaces.IContractModel) {
	contract.MakeAvailable()
}

func (service *ContractsService) HandleContractUpdated(price int, time int, hashrate int) {
	// contract.Save()
}

func (service *ContractsService) HandleDestinationUpdated(dest interfaces.IDestination) {
	// contract.Save()
}

func (service *ContractsService) HandleContractCreated(contract interfaces.IContractModel) {
	service.logger.Infof("created a contract %v", contract.GetId())

	contract.Save()
	service.logger.Debugf("Contract is available: %v", contract.IsAvailable())
	if contract.IsAvailable() {
		service.SubscribeToContractEvents(contract)
		// addr := service.blockchainGateway.HexToAddress(newContract.GetAddress())
		// hrLogs, hrSub, err := SubscribeToContractEvents(seller.EthClient, addr)
		// if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to events on hashrate contract %s, Fileline::%s, Error::", newContract.ID, lumerinlib.FileLine()), err)
		// }
		// go seller.WatchHashrateContract(addr.Hex(), hrLogs, hrSub)
	}
}

func (service *ContractsService) SubscribeToContractEvents(contract interfaces.IContractModel) error {
	service.logger.Debugf("Subscribing to blockchain gateway events for %v", contract.GetId())
	_, _, err := service.blockchainGateway.SubscribeToContractEvents(contract)

	if err != nil {
		return err
	}

	return nil
}

func (service *ContractsService) HandleContractPurchased(dest string, sellerAddress string, buyerAddress string) {
	// contract.Execute()
}

var _ interfaces.IContractsService = (*ContractsService)(nil)

func NewContractsService(
	logger interfaces.ILogger,
	validatorService interfaces.IValidatorsService,
	connectionsService interfaces.IConnectionsService,
	blockchainGateway interfaces.IBlockchainGateway,
	factory interfaces.IContractFactory,
	contractGateway interfaces.IContractsGateway,
	configuration *config.Config) interfaces.IContractsService {

	return &ContractsService{
		logger:             logger,
		validatorService:   validatorService,
		connectionsService: connectionsService,
		blockchainGateway:  blockchainGateway,
		factory:            factory,
		contractGateway:    contractGateway,
		configuration:      configuration,
	}
}

// func (service *ContractsService) HandleBuyerContractPurchased(args interfaces.IContractModel) {

// 	contractMsg := contract.(Contract)

// 	destMsg := msgbus.Dest{
// 		ID:     string(msgbus.GetRandomIDString()),
// 		NetUrl: string(destUrl),
// 	}
// 	buyer.Ps.PubWait(msgbus.DestMsg, string(destMsg.ID), destMsg)

// 	contractMsg.Dest = destMsg.ID
// 	contractMsg.State = ContRunningState
// 	buyer.Ps.PubWait(msgbus.ContractMsg, string(contractMsg.ID), contractMsg)

// 	buyer.NodeOperator.Contracts[contractMsg.ID] = ContRunningState
// 	buyer.Ps.SetWait(msgbus.NodeOperatorMsg, string(buyer.NodeOperator.ID), buyer.NodeOperator)
// }

// func (service *ContractsService) HandleBuyerContractUpdated(contract interfaces.IContractModel) {
// 	contract.Update()
// }

// func (service *ContractsService) HandleBuyerDestinationUpdated(args interfaces.IContractModel) {
// 	//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate Contract %s Cipher Text Updated \n\n", addr)

// 	event, err := buyer.Ps.GetWait(msgbus.ContractMsg, string(addr))
// 	if err != nil {
// 		//contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
// 	}
// 	if event.Err != nil {
// 		//contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
// 	}
// 	contractMsg := event.Data.(Contract)
// 	event, err = buyer.Ps.GetWait(msgbus.DestMsg, string(contractMsg.Dest))
// 	if err != nil {
// 		//contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Dest Failed: %v", err)
// 	}
// 	if event.Err != nil {
// 		//contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Dest Failed: %v", event.Err)
// 	}
// 	destMsg := event.Data.(msgbus.Dest)

// 	destUrl, err := readDestUrl(buyer.EthClient, common.HexToAddress(string(addr)), buyer.PrivateKey)
// 	if err != nil {
// 		//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
// 	}
// 	destMsg.NetUrl = string(destUrl)
// 	buyer.Ps.SetWait(msgbus.DestMsg, string(destMsg.ID), destMsg)
// }

// func (service *ContractsService) HandleBuyerContractClosed(args interfaces.IContractModel) {

// 	//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate Contract %s Closed \n\n", addr)

// 	buyer.Ps.Unpub(msgbus.ContractMsg, string(addr))

// 	delete(buyer.NodeOperator.Contracts, addr)
// 	buyer.Ps.SetWait(msgbus.NodeOperatorMsg, string(buyer.NodeOperator.ID), buyer.NodeOperator)
// }
