package contractmanager

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"gitlab.com/TitanInd/hashrouter/config"
	"gitlab.com/TitanInd/hashrouter/constants"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	"gitlab.com/TitanInd/lumerin/lumerinlib/clonefactory"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
	"gitlab.com/TitanInd/lumerin/lumerinlib/implementation"
)

const (
	AvailableState uint8 = 0
	RunningState   uint8 = 1
	HASHRATE_LIMIT       = 20
)

type ContractModel struct {
	Ps                    interfaces.IEventManager
	RoutableStreamService interfaces.IRoutableStreamService
	ContractGateway       interfaces.IContractGateway
	EthClient             *ethclient.Client
	CloneFactoryAddress   common.Address
	Account               common.Address
	PrivateKey            string
	CurrentNonce          nonce
	NodeOperator          *NodeOperator
	Ctx                   context.Context
	ClaimFunds            bool
	Logger                interfaces.ILogger
	//Buyer only
	TimeThreshold int
}

func readDestUrl(client *ethclient.Client, contractAddress common.Address, privateKeyString string) (string, error) {
	instance, err := implementation.NewImplementation(contractAddress, client)
	if err != nil {
		// fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return "", err
	}

	fmt.Printf("Getting Dest url from contract %s\n\n", contractAddress)

	encryptedDestUrl, err := instance.EncryptedPoolData(nil)
	if err != nil {
		// fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return "", err
	}

	/*
		// Decryption Logic
		destUrlBytes,_ := hex.DecodeString(encryptedDestUrl)
		PrivateKey, err := crypto.HexToECDSA(privateKeyString)
		if err != nil {
			log.Printf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
			return "", err
		}
		privateKeyECIES := ecies.ImportECDSA(PrivateKey)
		decryptedDestUrlBytes, err := privateKeyECIES.Decrypt(destUrlBytes, nil, nil)
		if err != nil {
			log.Printf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
			return "", err
		}
		decryptedDestUrl := string(decryptedDestUrlBytes)

		return decryptedDestUrl, err
	*/
	return encryptedDestUrl, err
}

func (seller *ContractModel) readContracts() ([]common.Address, error) {
	var sellerContractAddresses []common.Address
	var hashrateContractInstance *implementation.Implementation
	var hashrateContractSeller common.Address

	instance, err := clonefactory.NewClonefactory(seller.CloneFactoryAddress, seller.EthClient)
	if err != nil {
		// contextlib.Logf(seller.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		return sellerContractAddresses, err
	}

	hashrateContractAddresses, err := instance.GetContractList(&bind.CallOpts{})
	if err != nil {
		// contextlib.Logf(seller.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		return sellerContractAddresses, err
	}

	// parse existing hashrate contracts for ones that belong to seller
	for i := range hashrateContractAddresses {
		hashrateContractInstance, err = implementation.NewImplementation(hashrateContractAddresses[i], seller.EthClient)
		if err != nil {
			// contextlib.Logf(seller.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
			return sellerContractAddresses, err
		}
		hashrateContractSeller, err = hashrateContractInstance.Seller(nil)
		if err != nil {
			// contextlib.Logf(seller.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
			return sellerContractAddresses, err
		}
		if hashrateContractSeller == seller.Account {
			sellerContractAddresses = append(sellerContractAddresses, hashrateContractAddresses[i])
		}
	}

	return sellerContractAddresses, err
}

func (seller *ContractModel) watchContractCreation(cfLogs chan types.Log, cfSub ethereum.Subscription) {
	defer close(cfLogs)
	defer cfSub.Unsubscribe()

	// create event signature to parse out creation event
	contractCreatedSig := []byte("contractCreated(address,string)")
	contractCreatedSigHash := crypto.Keccak256Hash(contractCreatedSig)
	for {
		select {
		// case err := <-cfSub.Err():
		// contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		case <-seller.Ctx.Done():
			// contextlib.Logf(seller.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchContractCreation go routine")
			return
		case cfLog := <-cfLogs:
			if cfLog.Topics[0].Hex() == contractCreatedSigHash.Hex() {
				address := common.HexToAddress(cfLog.Topics[1].Hex())
				// check if contract created belongs to seller
				hashrateContractInstance, err := implementation.NewImplementation(address, seller.EthClient)
				if err != nil {
					seller.Logger.Error(err)
				}
				hashrateContractSeller, err := hashrateContractInstance.Seller(nil)
				if err != nil {
					seller.Logger.Error(err)
				}
				if hashrateContractSeller == seller.Account {
					seller.Logger.Info("Address of created Hashrate Contract: ", address.Hex())

					createdContractValues, err := seller.ReadHashrateContract(address)
					if err != nil {
						// contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					createdContractMsg := createContractMsg(address, createdContractValues, true)
					seller.Ps.PubWait(msgbus.ContractMsg, address.Hex(), createdContractMsg)

					seller.NodeOperator.Contracts[address.Hex()] = msgbus.ContAvailableState

					seller.Ps.SetWait(msgbus.NodeOperatorMsg, seller.NodeOperator.ID, seller.NodeOperator)
				}
			}
		}
	}
}

func (seller *ContractModel) watchHashrateContract(addr string, hrLogs chan types.Log, hrSub ethereum.Subscription) {
	contractEventChan := msgbus.NewEventChan()

	// check if contract is already in the running state and needs to be monitored for closeout
	event, err := seller.Ps.GetWait(msgbus.ContractMsg, addr)
	if err != nil {
		seller.Logger.Panicf("Getting Hashrate Contract Failed: %v", err)
	}
	if event.Err != nil {
		seller.Logger.Panicf("Getting Hashrate Contract Failed: %v", event.Err)
	}
	hashrateContractMsg := event.Data.(msgbus.Contract)
	if hashrateContractMsg.State == constants.ContRunningState {
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
			case err := <-hrSub.Err():
				contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
			case <-seller.Ctx.Done():
				contextlib.Logf(seller.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchHashrateContract go routine")
				return
			case hLog := <-hrLogs:
				switch hLog.Topics[0].Hex() {
				case contractPurchasedSigHash.Hex():
					buyer := common.HexToAddress(hLog.Topics[1].Hex())
					contextlib.Logf(seller.Ctx, log.LevelInfo, "%s purchased Hashrate Contract: %s\n\n", buyer.Hex(), addr)

					destUrl, err := readDestUrl(seller.EthClient, common.HexToAddress(string(addr)), seller.PrivateKey)
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					destMsg := msgbus.Dest{
						ID:     msgbus.DestID(msgbus.GetRandomIDString()),
						NetUrl: msgbus.DestNetUrl(destUrl),
					}
					seller.Ps.PubWait(msgbus.DestMsg, destMsg.ID, destMsg)

					event, err := seller.Ps.GetWait(msgbus.ContractMsg, addr)
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
					}
					if event.Err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
					}
					contractValues, err := seller.ReadHashrateContract(common.HexToAddress(string(addr)))
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					contractMsg := createContractMsg(common.HexToAddress(string(addr)), contractValues, true)
					contractMsg.Dest = destMsg.ID
					contractMsg.State = msgbus.ContRunningState
					contractMsg.Buyer = string(buyer.Hex())
					seller.Ps.SetWait(msgbus.ContractMsg, addr, contractMsg)

					seller.NodeOperator.Contracts[addr] = msgbus.ContRunningState
					seller.Ps.SetWait(msgbus.NodeOperatorMsg, seller.NodeOperator.ID, seller.NodeOperator)

				case cipherTextUpdatedSigHash.Hex():
					contextlib.Logf(seller.Ctx, log.LevelInfo, "Hashrate Contract %s Cipher Text Updated \n\n", addr)

					event, err := seller.Ps.GetWait(msgbus.ContractMsg, addr)
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
					}
					if event.Err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
					}
					contractMsg := event.Data.(msgbus.Contract)
					event, err = seller.Ps.GetWait(msgbus.DestMsg, contractMsg.Dest)
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Dest Failed: %v", err)
					}
					if event.Err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Dest Failed: %v", event.Err)
					}
					destMsg := event.Data.(msgbus.Dest)

					destUrl, err := readDestUrl(seller.EthClient, common.HexToAddress(string(addr)), seller.PrivateKey)
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					destMsg.NetUrl = msgbus.DestNetUrl(destUrl)
					seller.Ps.SetWait(msgbus.DestMsg, destMsg.ID, destMsg)

				case contractClosedSigHash.Hex():
					contextlib.Logf(seller.Ctx, log.LevelInfo, "Hashrate Contract %s Closed \n\n", addr)

					event, err := seller.Ps.GetWait(msgbus.ContractMsg, addr)
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
					}
					if event.Err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
					}
					contractMsg := event.Data.(msgbus.Contract)
					if contractMsg.State == msgbus.ContRunningState {
						contractMsg.State = msgbus.ContAvailableState
						contractMsg.Buyer = ""
						seller.Ps.SetWait(msgbus.ContractMsg, contractMsg.ID, contractMsg)

						seller.NodeOperator.Contracts[addr] = msgbus.ContAvailableState
						seller.Ps.SetWait(msgbus.NodeOperatorMsg, seller.NodeOperator.ID, seller.NodeOperator)
					}

				case purchaseInfoUpdatedSigHash.Hex():
					seller.Logger.Info("Hashrate Contract %s Purchase Info Updated \n\n", addr)

					event, err := seller.Ps.GetWait(msgbus.ContractMsg, addr)
					if err != nil {
						seller.Logger.Panicf("Getting Purchased Contract Failed: %v", err)
					}
					if event.Err != nil {
						seller.Logger.Panicf("Getting Purchased Contract Failed: %v", event.Err)
					}

					contractMsg := event.Data.(Contract)

					updatedContractValues, err := seller.ReadHashrateContract(common.HexToAddress(string(addr)))
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					updateContractMsg(&contractMsg, updatedContractValues)
					seller.Ps.SetWait(msgbus.ContractMsg, contractMsg.ID, contractMsg)
				}
			}
		}
	}()

	_, err = seller.Ps.Sub(msgbus.ContractMsg, addr, contractEventChan)
	if err != nil {
		contextlib.Logf(seller.Ctx, log.LevelPanic, "Subscribing to Contract Failed: %v", err)
	}
	// once contract is running, closeout after length of contract has passed if it was not closed out early
	for {
		select {
		case <-seller.Ctx.Done():
			contextlib.Logf(seller.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchHashrateContract go routine")
			return
		case event := <-contractEventChan:
			if event.EventType == msgbus.UpdateEvent {
				runningContractMsg := event.Data.(msgbus.Contract)
				if runningContractMsg.State == msgbus.ContRunningState {
					// run routine for each running contract to check if contract length has passed and contract should be closed out
					go seller.closeOutMonitor(runningContractMsg)
				}
			}
		}
	}
}

func (seller *ContractModel) closeOutMonitor(contractMsg Contract) {
	contractFinishedTimestamp := contractMsg.StartingBlockTimestamp + contractMsg.Length

	// subscribe to latest block headers
	headers := make(chan *types.Header)
	sub, err := seller.EthClient.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
	}
	defer close(headers)
	defer sub.Unsubscribe()

loop:
	for {
		select {
		case err := <-sub.Err():
			contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		case <-seller.Ctx.Done():
			contextlib.Logf(seller.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling closeout monitor go routine")
			return
		case header := <-headers:
			// get latest block from header
			block, err := seller.EthClient.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				contextlib.Logf(seller.Ctx, log.LevelWarn, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
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
					contractValues, err := seller.ReadHashrateContract(common.HexToAddress(string(contractMsg.ID)))
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					if contractValues.State == RunningState {
						var wg sync.WaitGroup
						wg.Add(1)
						err = setContractCloseOut(seller.EthClient, seller.Account, seller.PrivateKey, common.HexToAddress(string(contractMsg.ID)), &wg, &seller.CurrentNonce, closeOutType, seller.Ps, seller.NodeOperator)
						if err != nil {
							contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Contract Close Out failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
						}
						wg.Wait()
					}
					break loop
				}
			}

		}
	}
}

func (model *ContractModel) ReadHashrateContract(contractAddress common.Address) (hashrateContractValues, error) {
	var contractValues hashrateContractValues

	instance, err := implementation.NewImplementation(contractAddress, model.EthClient)

	if err != nil {
		model.Logger.Error(err)
		return contractValues, err
	}

	state, price, limit, speed, length, startingBlockTimestamp, buyer, seller, _, err := instance.GetPublicVariables(&bind.CallOpts{})
	if err != nil {
		model.Logger.Error(err)
		return contractValues, err
	}
	contractValues.State = state
	contractValues.Price = int(price.Int64())
	contractValues.Limit = int(limit.Int64())
	contractValues.Speed = int(speed.Int64())
	contractValues.Length = int(length.Int64())
	contractValues.StartingBlockTimestamp = int(startingBlockTimestamp.Int64())
	contractValues.Buyer = buyer
	contractValues.Seller = seller

	return contractValues, err
}
func createContractMsg(contractAddress common.Address, contractValues hashrateContractValues, isSeller bool) Contract {
	convertToMsgBusState := map[uint8]string{
		AvailableState: constants.ContAvailableState,
		RunningState:   constants.ContRunningState,
	}

	var contractMsg Contract
	contractMsg.IsSeller = isSeller
	contractMsg.ID = string(contractAddress.Hex())
	contractMsg.State = convertToMsgBusState[contractValues.State]
	contractMsg.Buyer = string(contractValues.Buyer.Hex())
	contractMsg.Price = contractValues.Price
	contractMsg.Limit = contractValues.Limit
	contractMsg.Speed = contractValues.Speed
	contractMsg.Length = contractValues.Length
	contractMsg.StartingBlockTimestamp = contractValues.StartingBlockTimestamp

	return contractMsg
}

func updateContractMsg(contractMsg *msgbus.Contract, contractValues hashrateContractValues) {
	contractMsg.Price = contractValues.Price
	contractMsg.Limit = contractValues.Limit
	contractMsg.Speed = contractValues.Speed
	contractMsg.Length = contractValues.Length
}

type NodeOperator struct {
	ID                     string
	IsBuyer                bool
	DefaultDest            string
	EthereumAccount        string
	TotalAvailableHashRate int
	UnusedHashRate         int
	Contracts              map[string]string
}

type Contract struct {
	IsSeller               bool
	ID                     string
	State                  string
	Buyer                  string
	Price                  int
	Limit                  int
	Speed                  int
	Length                 int
	StartingBlockTimestamp int
	Dest                   string
}

type hashrateContractValues struct {
	State                  uint8
	Price                  int
	Limit                  int
	Speed                  int
	Length                 int
	StartingBlockTimestamp int
	Buyer                  common.Address
	Seller                 common.Address
}

type nonce struct {
	mutex sync.Mutex
	nonce uint64
}

func NewContractModel(ctx context.Context, logger interfaces.ILogger, configuration *config.Config, eventManager interfaces.IEventManager, client *ethclient.Client, nodeOperator *NodeOperator, account *accounts.Account, privateKey string) *ContractModel {

	// var client *ethclient.Client
	// client, err = setUpClient(ethNodeAddr, seller.Account)
	// if err != nil {
	// 	return err
	// }
	contractConfig := configuration.Contract
	return &ContractModel{
		Ctx:        ctx,
		Ps:         eventManager,
		EthClient:  client,
		ClaimFunds: contractConfig.ClaimFunds,
		// ethNodeAddr:  contractConfig.EthNodeAddr,
		// mnemonic:     contractConfig.Mnemonic,
		// accountIndex: contractConfig.AccountIndex,
		Account:    account.Address,
		PrivateKey: privateKey,

		CloneFactoryAddress: common.HexToAddress(contractConfig.CloneFactoryAddress),

		NodeOperator: nodeOperator,
		Logger:       logger,
		// seller.NodeOperator.EthereumAccount = seller.Account.Hex(),
	}
}
