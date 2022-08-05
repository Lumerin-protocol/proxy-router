package contractmanager

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lumerinlib/implementation"
)

type ContractsService struct {
	logger interfaces.ILogger
}

func (service *ContractsService) CheckHashRate(contractId string) bool {
	// check for miners delivering hashrate for this contract
	totalHashrate := 0
	contractResult, err := service.GetContract(contractId)
	if err != nil {
		//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to get all miners, Fileline::%s, Error::", lumerinlib.FileLine()), err)
	}

	contract := contractResult

	// hashrate, err := buyer.Ps.GetHashrate()
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

	hashrateTolerance := float64(HASHRATE_LIMIT) / 100
	promisedHashrateMin := int(float64(contract.Speed) * (1 - hashrateTolerance))

	//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate being sent to contract %s: %d\n", contractId, totalHashrate)
	if totalHashrate <= promisedHashrateMin {
		//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Closing out contract %s for not meeting hashrate requirements\n", contractId)
		var wg sync.WaitGroup
		wg.Add(1)
		err := setContractCloseOut(buyer.EthClient, buyer.Account, buyer.PrivateKey, common.HexToAddress(string(contractId)), &wg, &buyer.CurrentNonce, 0, buyer.NodeOperator)
		if err != nil {
			//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Contract Close Out failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
		}
		wg.Wait()
		return true
	}
}

func (service *ContractsService) GetContract(contractId string) (interfaces.IContractModel, error) {
	return nil, nil
}

func (service *ContractsService) HandleContractClosed(contract interfaces.IContractModel) {

	//contextlib.Logf(seller.Ctx, log.LevelInfo, "Hashrate Contract %s Closed \n\n", addr)

	event, err := seller.Ps.GetWait(msgbus.ContractMsg, string(addr))
	if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
	}
	if event.Err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
	}
	contractMsg := event.Data.(Contract)
	if contractMsg.State == ContRunningState {
		contractMsg.State = ContAvailableState
		contractMsg.Buyer = ""
		seller.Ps.SetWait(msgbus.ContractMsg, string(contractMsg.ID), contractMsg)

		seller.NodeOperator.Contracts[addr] = ContAvailableState
		seller.Ps.SetWait(msgbus.NodeOperatorMsg, string(seller.NodeOperator.ID), seller.NodeOperator)
	}
}

func (service *ContractsService) HandleContractUpdated(contract interfaces.IContractModel) {
	//contextlib.Logf(seller.Ctx, log.LevelInfo, "Hashrate Contract %s Cipher Text Updated \n\n", addr)

	event, err := seller.Ps.GetWait(ContractMsg, string(addr))
	if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
	}
	if event.Err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
	}
	contractMsg := event.Data.(Contract)
	event, err = seller.Ps.GetWait(DestMsg, string(contractMsg.Dest))
	if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Dest Failed: %v", err)
	}
	if event.Err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Dest Failed: %v", event.Err)
	}
	destMsg := event.Data.(msgbus.Dest)

	seller.Ps.SetWait(msgbus.DestMsg, string(destMsg.ID), destMsg)
}

func (service *ContractsService) HandleContractCreated(contract interfaces.IContractModel) {

	address := common.HexToAddress(cfLog.Topics[1].Hex())
	// check if contract created belongs to seller
	hashrateContractInstance, err := implementation.NewImplementation(address, seller.EthClient)
	if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
	}
	hashrateContractSeller, err := hashrateContractInstance.Seller(nil)
	if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
	}
	if hashrateContractSeller == seller.Account {
		// TODO: Handle logs, errors and data
		//contextlib.Logf(seller.Ctx, log.LevelInfo, "Address of created Hashrate Contract: %s\n\n", address.Hex())

		// createdContractValues, err := readHashrateContract(seller.EthClient, address)
		// if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
		// }

		// createdContractMsg := createContractMsg(address, createdContractValues, true)
		// seller.Ps.PubWait(ContractMsg, string(address.Hex()), createdContractMsg)

		seller.NodeOperator.Contracts[string(address.Hex())] = ContAvailableState

		// seller.Ps.SetWait(NodeOperatorMsg, string(seller.NodeOperator.ID), seller.NodeOperator)
	}
}

func (service *ContractsService) HandleContractPurchased(contract interfaces.IContractModel) {
	// buyer := common.HexToAddress(hLog.Topics[1].Hex())
	//contextlib.Logf(seller.Ctx, log.LevelInfo, "%s purchased Hashrate Contract: %s\n\n", buyer.Hex(), addr)

	destUrl, err := readDestUrl(seller.EthClient, common.HexToAddress(string(addr)), seller.PrivateKey)
	if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
	}
	destMsg := Dest{
		ID:     string(uuid.NewString()),
		NetUrl: string(destUrl),
	}
	seller.Ps.PubWait(DestMsg, string(destMsg.ID), destMsg)

	event, err := seller.Ps.GetWait(ContractMsg, string(addr))
	if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
	}
	if event.Err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
	}
	contractValues, err := readHashrateContract(seller.EthClient, common.HexToAddress(string(addr)))
	if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
	}
	contractMsg := createContractMsg(common.HexToAddress(string(addr)), contractValues, true)
	contractMsg.Dest = destMsg.ID
	contractMsg.State = ContRunningState
	contractMsg.Buyer = string(buyer.Hex())
	seller.Ps.SetWait(ContractMsg, string(addr), contractMsg)

	seller.NodeOperator.Contracts[addr] = ContRunningState
	seller.Ps.SetWait(NodeOperatorMsg, string(seller.NodeOperator.ID), seller.NodeOperator)
}

func (service *ContractsService) HandleBuyerContractPurchased(args interfaces.IContractModel) {

	contractMsg := contract.(Contract)

	destMsg := msgbus.Dest{
		ID:     string(msgbus.GetRandomIDString()),
		NetUrl: string(destUrl),
	}
	buyer.Ps.PubWait(msgbus.DestMsg, string(destMsg.ID), destMsg)

	contractMsg.Dest = destMsg.ID
	contractMsg.State = ContRunningState
	buyer.Ps.PubWait(msgbus.ContractMsg, string(contractMsg.ID), contractMsg)

	buyer.NodeOperator.Contracts[contractMsg.ID] = ContRunningState
	buyer.Ps.SetWait(msgbus.NodeOperatorMsg, string(buyer.NodeOperator.ID), buyer.NodeOperator)
}

func (service *ContractsService) HandleBuyerContractUpdated(args interfaces.IContractModel) {
	//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate Contract %s Purchase Info Updated \n\n", addr)

	event, err := buyer.Ps.GetWait(msgbus.ContractMsg, string(addr))
	if err != nil {
		//contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
	}
	if event.Err != nil {
		//contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
	}
	contractMsg := event.Data.(Contract)

	updatedContractValues, err := readHashrateContract(buyer.EthClient, common.HexToAddress(string(addr)))
	if err != nil {
		//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
	}
	updateContractMsg(&contractMsg, updatedContractValues)
	buyer.Ps.SetWait(msgbus.ContractMsg, string(contractMsg.ID), contractMsg)
}

func (service *ContractsService) HandleBuyerDestinationUpdated(args interfaces.IContractModel) {
	//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate Contract %s Cipher Text Updated \n\n", addr)

	event, err := buyer.Ps.GetWait(msgbus.ContractMsg, string(addr))
	if err != nil {
		//contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
	}
	if event.Err != nil {
		//contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
	}
	contractMsg := event.Data.(Contract)
	event, err = buyer.Ps.GetWait(msgbus.DestMsg, string(contractMsg.Dest))
	if err != nil {
		//contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Dest Failed: %v", err)
	}
	if event.Err != nil {
		//contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Dest Failed: %v", event.Err)
	}
	destMsg := event.Data.(msgbus.Dest)

	destUrl, err := readDestUrl(buyer.EthClient, common.HexToAddress(string(addr)), buyer.PrivateKey)
	if err != nil {
		//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
	}
	destMsg.NetUrl = string(destUrl)
	buyer.Ps.SetWait(msgbus.DestMsg, string(destMsg.ID), destMsg)
}

func (service *ContractsService) HandleBuyerContractClosed(args interfaces.IContractModel) {

	//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate Contract %s Closed \n\n", addr)

	buyer.Ps.Unpub(msgbus.ContractMsg, string(addr))

	delete(buyer.NodeOperator.Contracts, addr)
	buyer.Ps.SetWait(msgbus.NodeOperatorMsg, string(buyer.NodeOperator.ID), buyer.NodeOperator)
}
