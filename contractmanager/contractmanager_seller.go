package contractmanager

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"gitlab.com/TitanInd/hashrouter/config"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
	"gitlab.com/TitanInd/hashrouter/lumerinlib/clonefactory"
	"gitlab.com/TitanInd/hashrouter/lumerinlib/implementation"
)

type SellerContractManager struct {
	ContractFactory     interfaces.IContractFactory
	Ps                  interfaces.IContractsService
	logger              interfaces.ILogger
	EthClient           *ethclient.Client
	CloneFactoryAddress interop.BlockchainAddress
	Account             interop.BlockchainAddress
	PrivateKey          string
	ClaimFunds          bool
	CurrentNonce        nonce
	NodeOperator        *NodeOperator
	Contracts           []Contract
	Ctx                 context.Context
}

func NewContractManager(
	ctx context.Context,
	contractsService interfaces.IContractsService,
	logger interfaces.ILogger,
	configuration *config.Config,
	eventManager interfaces.IEventManager,
	contractFactory interfaces.IContractFactory,
	client *ethclient.Client,
	nodeOperator *NodeOperator,
	wallet interfaces.IBlockchainWallet) (interfaces.ContractManager, error) {

	address, err := wallet.GetAddress()

	if err != nil {
		return nil, err
	}

	if configuration.Contract.IsBuyer {
		buyer := &BuyerContractManager{
			Logger:              logger,
			ContractFactory:     contractFactory,
			Ctx:                 ctx,
			Ps:                  contractsService,
			TimeThreshold:       configuration.Contract.TimeThreshold,
			EthClient:           client,
			CloneFactoryAddress: common.HexToAddress(configuration.Contract.Address),
			NodeOperator:        nodeOperator,
			PrivateKey:          wallet.GetPrivateKey(),
			Account:             address,
		}

		if buyer.NodeOperator.Contracts == nil {
			buyer.NodeOperator.Contracts = make(map[string]string)
		}

		return buyer, nil

	}

	seller := &SellerContractManager{
		logger:              logger,
		ContractFactory:     contractFactory,
		Ctx:                 ctx,
		Ps:                  contractsService,
		EthClient:           client,
		CloneFactoryAddress: common.HexToAddress(configuration.Contract.Address),
		NodeOperator:        nodeOperator,
		ClaimFunds:          configuration.Contract.ClaimFunds,
		PrivateKey:          wallet.GetPrivateKey(),
		Account:             address,
	}

	seller.NodeOperator.EthereumAccount = seller.Account.Hex()

	return seller, nil
}

func (seller *SellerContractManager) Run(ctx context.Context) (err error) {

	err = seller.SetupExistingContracts()
	if err != nil {
		return err
	}

	// routine for listensing to contract creation events that will update seller msg with new contracts and load new contract onto msgbus
	cfLogs, cfSub, err := SubscribeToContractEvents(seller.EthClient, seller.CloneFactoryAddress)
	if err != nil {
		return err
	}
	go seller.watchContractCreation(cfLogs, cfSub)

	// routine starts routines for seller's contracts that monitors contract purchase, close, and cancel events
	go func() {
		// start routines for existing contracts
		for addr := range seller.NodeOperator.Contracts {
			hrLogs, hrSub, err := SubscribeToContractEvents(seller.EthClient, common.HexToAddress(addr))
			if err != nil {
				//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to events on hashrate contract %s, Fileline::%s, Error::", addr, lumerinlib.FileLine()), err)
			}
			go seller.WatchHashrateContract(addr, hrLogs, hrSub)
		}

		// monitor new contracts getting created and start hashrate conrtract monitor routine when they are created
		seller.Ps.OnContractCreated(func(newContract interfaces.IContractModel) {

		})
	}()

	if err != nil {
		fmt.Printf("Error in start: %v\n", err)
	}

	return err
}

func (seller *SellerContractManager) SetupExistingContracts() (err error) {
	// var contractValues []hashrateContractValues
	var contractModels []interfaces.IContractModel

	sellerContracts, err := seller.ReadContracts()
	if err != nil {
		return err
	}
	//contextlib.Logf(seller.Ctx, log.LevelInfo, "Existing Seller Contracts: %v", sellerContracts)

	// get existing dests in msgbus to see if contract's dest already exists
	// existingDests := seller.Ps.GetDestinations()
	for _, sellerContract := range sellerContracts {
		id := string(sellerContract.Hex())
		if !seller.Ps.ContractExists(id) {

			contractMsg, err := readHashrateContract(seller.EthClient, sellerContract)

			if err != nil {
				return err
			}

			destUrl, err := readDestUrl(seller.EthClient, sellerContract, seller.PrivateKey)

			if err != nil {
				return err
			}

			contract, err := seller.ContractFactory.CreateContract(true, sellerContract.Hex(), ContractStateEnum[contractMsg.State], contractMsg.Buyer.Hex(), contractMsg.Price, contractMsg.Limit, contractMsg.Speed, contractMsg.Length, contractMsg.StartingBlockTimestamp, destUrl)

			if err != nil {
				return err
			}

			contractModels = append(contractModels, contract)

			contract, err = contract.TryRunningAt(destUrl)

			if err != nil {
				return err
			}
		}
	}

	_, err = seller.Ps.SaveContracts(contractModels)

	return err
}

func (seller *SellerContractManager) ReadContracts() ([]interop.BlockchainAddress, error) {
	var sellerContractAddresses []interop.BlockchainAddress
	var hashrateContractInstance *implementation.Implementation
	var hashrateContractSeller interop.BlockchainAddress

	seller.logger.Infof("instantiating clonefactory %v", seller.CloneFactoryAddress)
	instance, err := clonefactory.NewClonefactory(seller.CloneFactoryAddress, seller.EthClient)
	if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		return sellerContractAddresses, err
	}

	hashrateContractAddresses, err := instance.GetContractList(&bind.CallOpts{})
	if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		return sellerContractAddresses, err
	}

	// parse existing hashrate contracts for ones that belong to seller
	for i := range hashrateContractAddresses {
		hashrateContractInstance, err = implementation.NewImplementation(hashrateContractAddresses[i], seller.EthClient)
		if err != nil {
			//contextlib.Logf(seller.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
			return sellerContractAddresses, err
		}
		hashrateContractSeller, err = hashrateContractInstance.Seller(nil)
		if err != nil {
			//contextlib.Logf(seller.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
			return sellerContractAddresses, err
		}
		if hashrateContractSeller == seller.Account {
			sellerContractAddresses = append(sellerContractAddresses, hashrateContractAddresses[i])
		}
	}

	return sellerContractAddresses, err
}

func (seller *SellerContractManager) watchContractCreation(cfLogs chan types.Log, cfSub ethereum.Subscription) {
	defer close(cfLogs)
	defer cfSub.Unsubscribe()

	// create event signature to parse out creation event
	contractCreatedSig := []byte("contractCreated(address,string)")
	contractCreatedSigHash := crypto.Keccak256Hash(contractCreatedSig)
	for {
		select {
		// TODO: handle errors
		case err := <-cfSub.Err():
			seller.logger.Errorf("%v", err)
		// contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		case <-seller.Ctx.Done():
			//contextlib.Logf(seller.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchContractCreation go routine")
			return
		case cfLog := <-cfLogs:
			if cfLog.Topics[0].Hex() == contractCreatedSigHash.Hex() {
				address := common.HexToAddress(cfLog.Topics[1].Hex())

				seller.logger.Debugf("contract created: %v", address)
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

					createdContractValues, err := readHashrateContract(seller.EthClient, address)
					if err != nil {
						//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}

					createdContractMsg := createContractMsg(address, createdContractValues, true)
					// seller.Ps.PubWait(ContractMsg, string(address.Hex()), createdContractMsg)

					seller.NodeOperator.Contracts[string(address.Hex())] = ContAvailableState

					// seller.Ps.SetWait(NodeOperatorMsg, string(seller.NodeOperator.ID), seller.NodeOperator)
					seller.Ps.HandleContractCreated(createdContractMsg)
				}
			}
		}
	}
}

func (seller *SellerContractManager) WatchHashrateContract(addr string, hrLogs chan types.Log, hrSub ethereum.Subscription) {

	// check if contract is already in the running state and needs to be monitored for closeout
	contract, err := seller.Ps.GetContract(addr)
	if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Hashrate Contract Failed: %v", err)
	}

	hashrateContractMsg := contract.(*Contract)
	if hashrateContractMsg.State == ContRunningState {
		go seller.closeOutMonitor(hashrateContractMsg)
	}

	// create event signatures to parse out which event was being emitted from hashrate contract
	contractPurchasedSig := []byte("contractPurchased(address)")
	contractClosedSig := []byte("contractClosed()")
	purchaseInfoUpdatedSig := []byte("purchaseInfoUpdated()")
	cipherTextUpdatedSig := []byte("cipherTextUpdated(string)")
	contractPurchasedSigHash := crypto.Keccak256Hash(contractPurchasedSig)
	contractClosedSigHash := crypto.Keccak256Hash(contractClosedSig)
	purchaseInfoUpdatedSigHash := crypto.Keccak256Hash(purchaseInfoUpdatedSig)
	cipherTextUpdatedSigHash := crypto.Keccak256Hash(cipherTextUpdatedSig)

	// routine monitoring and acting upon events emmited by hashrate contract
	go func() {
		defer close(hrLogs)
		defer hrSub.Unsubscribe()
		for {
			select {
			// TODO: handle errors
			// case err := <-hrSub.Err():
			//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
			case <-seller.Ctx.Done():
				//contextlib.Logf(seller.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchHashrateContract go routine")
				return
			case hLog := <-hrLogs:

				destUrl, err := readDestUrl(seller.EthClient, common.HexToAddress(string(addr)), seller.PrivateKey)

				if err != nil {
					//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
				}

				hashrateContractMsg.Dest = destUrl

				switch hLog.Topics[0].Hex() {
				case contractPurchasedSigHash.Hex():
					destUrl, err := readDestUrl(seller.EthClient, common.HexToAddress(string(addr)), seller.PrivateKey)

					if err != nil {
						//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					buyer := common.HexToAddress(hLog.Topics[1].Hex())
					hashrateContractMsg.Dest = destUrl
					hashrateContractMsg.Buyer = string(buyer.Hex())

					seller.Ps.HandleContractPurchased(hashrateContractMsg)

				case cipherTextUpdatedSigHash.Hex():

					destUrl, err := readDestUrl(seller.EthClient, common.HexToAddress(string(addr)), seller.PrivateKey)

					if err != nil {
						//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}

					hashrateContractMsg.Dest = destUrl

					seller.Ps.HandleDestinationUpdated(hashrateContractMsg)

				case contractClosedSigHash.Hex():
					seller.Ps.HandleContractClosed(hashrateContractMsg)

				case purchaseInfoUpdatedSigHash.Hex():
					updatedContractValues, err := readHashrateContract(seller.EthClient, common.HexToAddress(string(addr)))

					if err != nil {
						//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}

					updateContractMsg(hashrateContractMsg, updatedContractValues)

					seller.Ps.HandleContractUpdated(hashrateContractMsg)

				}
			}
		}
	}()

	// TODO: Handle Update event
	// _, err = seller.Ps.Sub(msgbus.ContractMsg, string(addr), contractEventChan)
	// if err != nil {
	//contextlib.Logf(seller.Ctx, log.LevelPanic, "Subscribing to Contract Failed: %v", err)
	// }
	// once contract is running, closeout after length of contract has passed if it was not closed out early
	// for {
	// 	select {
	// 	case <-seller.Ctx.Done():
	// 		//contextlib.Logf(seller.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchHashrateContract go routine")
	// 		return
	// 	case event := <-contractEventChan:
	// if event.string == msgbus.UpdateEvent {
	// 	runningContractMsg := event.Data.(Contract)
	// 	if runningContractMsg.State == ContRunningState {
	// 		// run routine for each running contract to check if contract length has passed and contract should be closed out
	// 		go seller.closeOutMonitor(runningContractMsg)
	// 	}
	// }
	// 	}
	// }
}

func (seller *SellerContractManager) closeOutMonitor(contractMsg *Contract) {
	contractFinishedTimestamp := contractMsg.StartingBlockTimestamp + contractMsg.Length

	// subscribe to latest block headers
	headers := make(chan *types.Header)
	sub, err := seller.EthClient.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
	}
	defer close(headers)
	defer sub.Unsubscribe()

loop:
	for {
		select {
		//TODO: handle errors
		// case err := <-sub.Err():
		//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		case <-seller.Ctx.Done():
			//contextlib.Logf(seller.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling closeout monitor go routine")
			return
		case header := <-headers:
			// get latest block from header
			block, err := seller.EthClient.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				//contextlib.Logf(seller.Ctx, log.LevelWarn, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
			} else {
				// check if contract length has passed
				if block.Time() >= uint64(contractFinishedTimestamp) {
					var closeOutType uint

					// seller only wants to closeout
					closeOutType = 2
					// seller wants to claim funds with closeout
					if seller.ClaimFunds {
						closeOutType = 3
					}

					// if contract was not already closed early, close out here
					contractValues, err := readHashrateContract(seller.EthClient, common.HexToAddress(string(contractMsg.ID)))
					if err != nil {
						//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					if contractValues.State == RunningState {
						var wg sync.WaitGroup
						wg.Add(1)
						err = setContractCloseOut(seller.EthClient, seller.Account, seller.PrivateKey, common.HexToAddress(string(contractMsg.ID)), &wg, &seller.CurrentNonce, closeOutType, seller.NodeOperator)
						if err != nil {
							//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Contract Close Out failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
						}
						wg.Wait()
					}
					break loop
				}
			}

		}
	}
}
