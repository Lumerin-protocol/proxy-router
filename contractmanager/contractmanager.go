package contractmanager

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"
	//"encoding/hex"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	//"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/ethclient"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"

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
	HASHRATE_LIMIT = 20
)

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

type ContractManager interface {
	start() (err error)
	init(Ctx *context.Context, contractManagerConfigID msgbus.IDString, nodeOperatorMsg *msgbus.NodeOperator) (err error)
	setupExistingContracts() (err error)
	readContracts() ([]common.Address, error)
	watchHashrateContract(addr msgbus.ContractID, hrLogs chan types.Log, hrSub ethereum.Subscription)
}

type SellerContractManager struct {
	Ps                  *msgbus.PubSub
	EthClient           *ethclient.Client
	CloneFactoryAddress common.Address
	Account             common.Address
	PrivateKey          string
	ClaimFunds          bool
	CurrentNonce        nonce
	NodeOperator        msgbus.NodeOperator
	Ctx                 context.Context
}

type BuyerContractManager struct {
	Ps                  *msgbus.PubSub
	EthClient           *ethclient.Client
	CloneFactoryAddress common.Address
	Account             common.Address
	PrivateKey          string
	CurrentNonce        nonce
	TimeThreshold       int
	NodeOperator        msgbus.NodeOperator
	Ctx                 context.Context
}

func Run(Ctx *context.Context, contractManager ContractManager, contractManagerConfigID msgbus.IDString, nodeOperatorMsg *msgbus.NodeOperator) (err error) {
	contractManagerCtx, contractManagerCancel := context.WithCancel(*Ctx)
	go newConfigMonitor(Ctx, contractManagerCtx, contractManagerCancel, contractManager, contractManagerConfigID, nodeOperatorMsg)

	err = contractManager.init(&contractManagerCtx, contractManagerConfigID, nodeOperatorMsg)
	if err != nil {
		return err
	}
	err = contractManager.start()
	if err != nil {
		return err
	}

	return err
}

func newConfigMonitor(Ctx *context.Context, contractManagerCtx context.Context, contractManagerCancel context.CancelFunc, contractManager ContractManager, contractManagerConfigID msgbus.IDString, nodeOperatorMsg *msgbus.NodeOperator) {
	contractConfigCh := msgbus.NewEventChan()
	cs := contextlib.GetContextStruct(contractManagerCtx)
	Ps := cs.MsgBus

	event, err := Ps.SubWait(msgbus.ContractManagerConfigMsg, contractManagerConfigID, contractConfigCh)
	if err != nil {
		contextlib.Logf(contractManagerCtx, log.LevelPanic, "SubWait failed: %v", err)
	}
	if event.EventType != msgbus.SubscribedEvent {
		contextlib.Logf(contractManagerCtx, log.LevelPanic, "Wrong event type: %v", err)
	}

	for event = range contractConfigCh {
		if event.EventType == msgbus.UpdateEvent {
			contextlib.Logf(contractManagerCtx, log.LevelInfo, "Updated Contract Manager Configuration: Restarting Contract Manager: %v\n", event)
			contractManagerCancel()
			err = Run(Ctx, contractManager, contractManagerConfigID, nodeOperatorMsg)
			if err != nil {
				contextlib.Logf(contractManagerCtx, log.LevelPanic, "Contract manager failed to run: %v", err)
			}
			return
		}
	}
}

func (seller *SellerContractManager) init(Ctx *context.Context, contractManagerConfigID msgbus.IDString, nodeOperatorMsg *msgbus.NodeOperator) (err error) {
	seller.Ctx = *Ctx
	cs := contextlib.GetContextStruct(seller.Ctx)
	seller.Ps = cs.MsgBus

	event, err := seller.Ps.GetWait(msgbus.ContractManagerConfigMsg, contractManagerConfigID)
	if err != nil {
		return err
	}
	contractManagerConfig := event.Data.(msgbus.ContractManagerConfig)
	seller.ClaimFunds = contractManagerConfig.ClaimFunds
	ethNodeAddr := contractManagerConfig.EthNodeAddr
	mnemonic := contractManagerConfig.Mnemonic
	accountIndex := contractManagerConfig.AccountIndex

	Account, PrivateKey := HdWalletKeys(mnemonic, accountIndex)
	seller.Account = Account.Address
	seller.PrivateKey = PrivateKey

	var client *ethclient.Client
	client, err = setUpClient(ethNodeAddr, seller.Account)
	if err != nil {
		return err
	}
	seller.EthClient = client
	seller.CloneFactoryAddress = common.HexToAddress(contractManagerConfig.CloneFactoryAddress)

	seller.NodeOperator = *nodeOperatorMsg
	seller.NodeOperator.EthereumAccount = seller.Account.Hex()

	return err
}

func (seller *SellerContractManager) start() (err error) {
	err = seller.setupExistingContracts()
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
			hrLogs, hrSub, err := SubscribeToContractEvents(seller.EthClient, common.HexToAddress(string(addr)))
			if err != nil {
				contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to events on hashrate contract %s, Fileline::%s, Error::", addr, lumerinlib.FileLine()), err)
			}
			go seller.watchHashrateContract(addr, hrLogs, hrSub)
		}

		// monitor new contracts getting created and start hashrate conrtract monitor routine when they are created
		contractEventChan := msgbus.NewEventChan()
		_, err = seller.Ps.Sub(msgbus.ContractMsg, "", contractEventChan)
		if err != nil {
			contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to contract events on msgbus, Fileline::%s, Error::", lumerinlib.FileLine()), err)
		}
		for {
			select {
			case <-seller.Ctx.Done():
				contextlib.Logf(seller.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling start routine")
				return
			case event := <-contractEventChan:
				if event.EventType == msgbus.PublishEvent {
					newContract := event.Data.(msgbus.Contract)
					if newContract.State == msgbus.ContAvailableState {
						addr := common.HexToAddress(string(newContract.ID))
						hrLogs, hrSub, err := SubscribeToContractEvents(seller.EthClient, addr)
						if err != nil {
							contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to events on hashrate contract %s, Fileline::%s, Error::", newContract.ID, lumerinlib.FileLine()), err)
						}
						go seller.watchHashrateContract(msgbus.ContractID(addr.Hex()), hrLogs, hrSub)
					}
				}
			}
		}
	}()
	fmt.Printf("Error in start: %v\n", err)

	return err
}

func (seller *SellerContractManager) setupExistingContracts() (err error) {
	var contractValues []hashrateContractValues
	var contractMsgs []msgbus.Contract

	sellerContracts, err := seller.readContracts()
	if err != nil {
		return err
	}
	contextlib.Logf(seller.Ctx, log.LevelInfo, "Existing Seller Contracts: %v", sellerContracts)

	for i := range sellerContracts {
		id := msgbus.ContractID(sellerContracts[i].Hex())
		if _, ok := seller.NodeOperator.Contracts[id]; !ok {
			contract, err := readHashrateContract(seller.EthClient, sellerContracts[i])
			if err != nil {
				return err
			}
			contractValues = append(contractValues, contract)
			contractMsgs = append(contractMsgs, createContractMsg(sellerContracts[i], contractValues[i], true))

			seller.NodeOperator.Contracts[msgbus.ContractID(sellerContracts[i].Hex())] = msgbus.ContAvailableState

			if contractValues[i].State == RunningState {
				seller.NodeOperator.Contracts[msgbus.ContractID(sellerContracts[i].Hex())] = msgbus.ContRunningState

				// get existing dests in msgbus to see if contract's dest already exists
				event, err := seller.Ps.GetWait(msgbus.DestMsg, "")
				if err != nil {
					contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting existing dests Failed: %v", err)
				}
				existingDests := event.Data.(msgbus.IDIndex)

				destUrl, err := readDestUrl(seller.EthClient, sellerContracts[i], seller.PrivateKey)
				if err != nil {
					contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
				}

				// if msgbus has dest with same target address, use that as contract msg dest
				for _, v := range existingDests {
					existingDest, err := seller.Ps.DestGetWait(msgbus.DestID(v))
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting existing dest Failed: %v", err)
					}
					if existingDest.NetUrl == msgbus.DestNetUrl(destUrl) {
						contractMsgs[i].Dest = msgbus.DestID(v)
					}
				}

				// msgbus does not have dest with that target address
				if contractMsgs[i].Dest == "" {
					destMsg := msgbus.Dest{
						ID:     msgbus.DestID(msgbus.GetRandomIDString()),
						NetUrl: msgbus.DestNetUrl(destUrl),
					}
					seller.Ps.PubWait(msgbus.DestMsg, msgbus.IDString(destMsg.ID), destMsg)

					contractMsgs[i].Dest = destMsg.ID
				}
			}

			seller.Ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contractMsgs[i].ID), contractMsgs[i])
		}
	}

	seller.Ps.SetWait(msgbus.NodeOperatorMsg, msgbus.IDString(seller.NodeOperator.ID), seller.NodeOperator)

	return err
}

func (seller *SellerContractManager) readContracts() ([]common.Address, error) {
	var sellerContractAddresses []common.Address
	var hashrateContractInstance *implementation.Implementation
	var hashrateContractSeller common.Address

	instance, err := clonefactory.NewClonefactory(seller.CloneFactoryAddress, seller.EthClient)
	if err != nil {
		contextlib.Logf(seller.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		return sellerContractAddresses, err
	}

	hashrateContractAddresses, err := instance.GetContractList(&bind.CallOpts{})
	if err != nil {
		contextlib.Logf(seller.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		return sellerContractAddresses, err
	}

	// parse existing hashrate contracts for ones that belong to seller
	for i := range hashrateContractAddresses {
		hashrateContractInstance, err = implementation.NewImplementation(hashrateContractAddresses[i], seller.EthClient)
		if err != nil {
			contextlib.Logf(seller.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
			return sellerContractAddresses, err
		}
		hashrateContractSeller, err = hashrateContractInstance.Seller(nil)
		if err != nil {
			contextlib.Logf(seller.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
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
		case err := <-cfSub.Err():
			contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		case <-seller.Ctx.Done():
			contextlib.Logf(seller.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchContractCreation go routine")
			return
		case cfLog := <-cfLogs:
			if cfLog.Topics[0].Hex() == contractCreatedSigHash.Hex() {
				address := common.HexToAddress(cfLog.Topics[1].Hex())
				// check if contract created belongs to seller
				hashrateContractInstance, err := implementation.NewImplementation(address, seller.EthClient)
				if err != nil {
					contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
				}
				hashrateContractSeller, err := hashrateContractInstance.Seller(nil)
				if err != nil {
					contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
				}
				if hashrateContractSeller == seller.Account {
					contextlib.Logf(seller.Ctx, log.LevelInfo, "Address of created Hashrate Contract: %s\n\n", address.Hex())

					createdContractValues, err := readHashrateContract(seller.EthClient, address)
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					createdContractMsg := createContractMsg(address, createdContractValues, true)
					seller.Ps.PubWait(msgbus.ContractMsg, msgbus.IDString(address.Hex()), createdContractMsg)

					seller.NodeOperator.Contracts[msgbus.ContractID(address.Hex())] = msgbus.ContAvailableState

					seller.Ps.SetWait(msgbus.NodeOperatorMsg, msgbus.IDString(seller.NodeOperator.ID), seller.NodeOperator)
				}
			}
		}
	}
}

func (seller *SellerContractManager) watchHashrateContract(addr msgbus.ContractID, hrLogs chan types.Log, hrSub ethereum.Subscription) {
	contractEventChan := msgbus.NewEventChan()

	// check if contract is already in the running state and needs to be monitored for closeout
	event, err := seller.Ps.GetWait(msgbus.ContractMsg, msgbus.IDString(addr))
	if err != nil {
		contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Hashrate Contract Failed: %v", err)
	}
	if event.Err != nil {
		contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Hashrate Contract Failed: %v", event.Err)
	}
	hashrateContractMsg := event.Data.(msgbus.Contract)
	if hashrateContractMsg.State == msgbus.ContRunningState {
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
					seller.Ps.PubWait(msgbus.DestMsg, msgbus.IDString(destMsg.ID), destMsg)

					event, err := seller.Ps.GetWait(msgbus.ContractMsg, msgbus.IDString(addr))
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
					}
					if event.Err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
					}
					contractValues, err := readHashrateContract(seller.EthClient, common.HexToAddress(string(addr)))
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					contractMsg := createContractMsg(common.HexToAddress(string(addr)), contractValues, true)
					contractMsg.Dest = destMsg.ID
					contractMsg.State = msgbus.ContRunningState
					contractMsg.Buyer = string(buyer.Hex())
					seller.Ps.SetWait(msgbus.ContractMsg, msgbus.IDString(addr), contractMsg)

					seller.NodeOperator.Contracts[addr] = msgbus.ContRunningState
					seller.Ps.SetWait(msgbus.NodeOperatorMsg, msgbus.IDString(seller.NodeOperator.ID), seller.NodeOperator)

				case cipherTextUpdatedSigHash.Hex():
					contextlib.Logf(seller.Ctx, log.LevelInfo, "Hashrate Contract %s Cipher Text Updated \n\n", addr)

					event, err := seller.Ps.GetWait(msgbus.ContractMsg, msgbus.IDString(addr))
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
					}
					if event.Err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
					}
					contractMsg := event.Data.(msgbus.Contract)
					event, err = seller.Ps.GetWait(msgbus.DestMsg, msgbus.IDString(contractMsg.Dest))
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
					seller.Ps.SetWait(msgbus.DestMsg, msgbus.IDString(destMsg.ID), destMsg)

				case contractClosedSigHash.Hex():
					contextlib.Logf(seller.Ctx, log.LevelInfo, "Hashrate Contract %s Closed \n\n", addr)

					event, err := seller.Ps.GetWait(msgbus.ContractMsg, msgbus.IDString(addr))
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
						seller.Ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contractMsg.ID), contractMsg)
	
						seller.NodeOperator.Contracts[addr] = msgbus.ContAvailableState
						seller.Ps.SetWait(msgbus.NodeOperatorMsg, msgbus.IDString(seller.NodeOperator.ID), seller.NodeOperator)
					}

				case purchaseInfoUpdatedSigHash.Hex():
					contextlib.Logf(seller.Ctx, log.LevelInfo, "Hashrate Contract %s Purchase Info Updated \n\n", addr)

					event, err := seller.Ps.GetWait(msgbus.ContractMsg, msgbus.IDString(addr))
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
					}
					if event.Err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
					}
					contractMsg := event.Data.(msgbus.Contract)

					updatedContractValues, err := readHashrateContract(seller.EthClient, common.HexToAddress(string(addr)))
					if err != nil {
						contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					updateContractMsg(&contractMsg, updatedContractValues)
					seller.Ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contractMsg.ID), contractMsg)
				}
			}
		}
	}()

	_, err = seller.Ps.Sub(msgbus.ContractMsg, msgbus.IDString(addr), contractEventChan)
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

func (seller *SellerContractManager) closeOutMonitor(contractMsg msgbus.Contract) {
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
					contractValues, err := readHashrateContract(seller.EthClient, common.HexToAddress(string(contractMsg.ID)))
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

func (buyer *BuyerContractManager) init(Ctx *context.Context, contractManagerConfigID msgbus.IDString, nodeOperatorMsg *msgbus.NodeOperator) (err error) {
	buyer.Ctx = *Ctx
	cs := contextlib.GetContextStruct(buyer.Ctx)
	buyer.Ps = cs.MsgBus

	event, err := buyer.Ps.GetWait(msgbus.ContractManagerConfigMsg, contractManagerConfigID)
	if err != nil {
		return err
	}
	contractManagerConfig := event.Data.(msgbus.ContractManagerConfig)
	buyer.TimeThreshold = contractManagerConfig.TimeThreshold
	ethNodeAddr := contractManagerConfig.EthNodeAddr
	mnemonic := contractManagerConfig.Mnemonic
	accountIndex := contractManagerConfig.AccountIndex

	Account, PrivateKey := HdWalletKeys(mnemonic, accountIndex)
	buyer.Account = Account.Address
	buyer.PrivateKey = PrivateKey

	var client *ethclient.Client
	client, err = setUpClient(ethNodeAddr, buyer.Account)
	if err != nil {
		return err
	}
	buyer.EthClient = client
	buyer.CloneFactoryAddress = common.HexToAddress(contractManagerConfig.CloneFactoryAddress)

	buyer.NodeOperator = *nodeOperatorMsg
	buyer.NodeOperator.EthereumAccount = buyer.Account.Hex()

	if buyer.NodeOperator.Contracts == nil {
		buyer.NodeOperator.Contracts = make(map[msgbus.ContractID]msgbus.ContractState)
	}

	return err
}

func (buyer *BuyerContractManager) start() (err error) {
	err = buyer.setupExistingContracts()
	if err != nil {
		return err
	}

	// routine for listensing to contract purchase events to update buyer with new contracts they purchased
	cfLogs, cfSub, err := SubscribeToContractEvents(buyer.EthClient, buyer.CloneFactoryAddress)
	if err != nil {
		return err
	}
	go buyer.watchContractPurchase(cfLogs, cfSub)

	// miner event channel for miner monitor that checks miner publishes, updates, and deletes
	minerEventChan := msgbus.NewEventChan()
	_, err = buyer.Ps.Sub(msgbus.MinerMsg, msgbus.IDString(""), minerEventChan)
	if err != nil {
		contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to miner events on msgbus, Fileline::%s, Error::", lumerinlib.FileLine()), err)
	}

	// routine starts routines for buyers's contracts that monitors contract running and close events
	go func() {
		// start watch hashrate contract for existing running contracts
		for addr := range buyer.NodeOperator.Contracts {
			hrLogs, hrSub, err := SubscribeToContractEvents(buyer.EthClient, common.HexToAddress(string(addr)))
			if err != nil {
				contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to events on hashrate contract %s, Fileline::%s, Error::", addr, lumerinlib.FileLine()), err)
			}
			go buyer.watchHashrateContract(addr, hrLogs, hrSub)

			contractEventChan := msgbus.NewEventChan()
			_, err = buyer.Ps.Sub(msgbus.ContractMsg, msgbus.IDString(addr), contractEventChan)
			if err != nil {
				contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to contract %s events, Fileline::%s, Error::", addr, lumerinlib.FileLine()), err)
			}
			go buyer.closeOutMonitor(minerEventChan, contractEventChan, addr)
		}

		// monitor new contracts getting purchased and start watch hashrate conrtract routine when they are purchased
		contractEventChan := msgbus.NewEventChan()
		_, err := buyer.Ps.Sub(msgbus.ContractMsg, "", contractEventChan)
		if err != nil {
			contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to contract events on msgbus, Fileline::%s, Error::", lumerinlib.FileLine()), err)
		}
		for {
			select {
			case <-buyer.Ctx.Done():
				contextlib.Logf(buyer.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling start routine")
				return
			case event := <-contractEventChan:
				if event.EventType == msgbus.PublishEvent {
					newContract := event.Data.(msgbus.Contract)
					addr := common.HexToAddress(string(newContract.ID))
					hrLogs, hrSub, err := SubscribeToContractEvents(buyer.EthClient, addr)
					if err != nil {
						contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to events on hashrate contract %s, Fileline::%s, Error::", addr, lumerinlib.FileLine()), err)
					}
					go buyer.watchHashrateContract(msgbus.ContractID(addr.Hex()), hrLogs, hrSub)

					newContractEventChan := msgbus.NewEventChan()
					_, err = buyer.Ps.Sub(msgbus.ContractMsg, msgbus.IDString(newContract.ID), newContractEventChan)
					if err != nil {
						contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to contract %s events, Fileline::%s, Error::", newContract.ID, lumerinlib.FileLine()), err)
					}
					go buyer.closeOutMonitor(minerEventChan, newContractEventChan, newContract.ID)
				}
			}
		}
	}()
	return nil
}

func (buyer *BuyerContractManager) setupExistingContracts() (err error) {
	var contractValues []hashrateContractValues
	var contractMsgs []msgbus.Contract
	var nodeOperatorUpdated bool

	buyerContracts, err := buyer.readContracts()
	if err != nil {
		return err
	}
	contextlib.Logf(buyer.Ctx, log.LevelInfo, "Existing Buyer Contracts: %v", buyerContracts)

	for i := range buyerContracts {
		id := msgbus.ContractID(buyerContracts[i].Hex())
		if _, ok := buyer.NodeOperator.Contracts[id]; !ok {
			contract, err := readHashrateContract(buyer.EthClient, buyerContracts[i])
			if err != nil {
				return err
			}
			contractValues = append(contractValues, contract)
			contractMsgs = append(contractMsgs, createContractMsg(buyerContracts[i], contractValues[i], false))
			buyer.Ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contractMsgs[i].ID), contractMsgs[i])

			buyer.NodeOperator.Contracts[msgbus.ContractID(buyerContracts[i].Hex())] = msgbus.ContRunningState
			nodeOperatorUpdated = true
		}
	}

	if nodeOperatorUpdated {
		buyer.Ps.PubWait(msgbus.NodeOperatorMsg, msgbus.IDString(buyer.NodeOperator.ID), buyer.NodeOperator)
	}

	return err
}

func (buyer *BuyerContractManager) readContracts() ([]common.Address, error) {
	var buyerContractAddresses []common.Address
	var hashrateContractInstance *implementation.Implementation
	var hashrateContractBuyer common.Address

	instance, err := clonefactory.NewClonefactory(buyer.CloneFactoryAddress, buyer.EthClient)
	if err != nil {
		contextlib.Logf(buyer.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		return buyerContractAddresses, err
	}

	hashrateContractAddresses, err := instance.GetContractList(&bind.CallOpts{})
	if err != nil {
		contextlib.Logf(buyer.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		return buyerContractAddresses, err
	}

	// parse existing hashrate contracts for ones that belong to buyer
	for i := range hashrateContractAddresses {
		hashrateContractInstance, err = implementation.NewImplementation(hashrateContractAddresses[i], buyer.EthClient)
		if err != nil {
			contextlib.Logf(buyer.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
			return buyerContractAddresses, err
		}
		hashrateContractBuyer, err = hashrateContractInstance.Buyer(nil)
		if err != nil {
			contextlib.Logf(buyer.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
			return buyerContractAddresses, err
		}
		if hashrateContractBuyer == buyer.Account {
			buyerContractAddresses = append(buyerContractAddresses, hashrateContractAddresses[i])
		}
	}

	return buyerContractAddresses, err
}

func (buyer *BuyerContractManager) watchContractPurchase(cfLogs chan types.Log, cfSub ethereum.Subscription) {
	defer close(cfLogs)
	defer cfSub.Unsubscribe()

	// create event signature to parse out purchase event
	clonefactoryContractPurchasedSig := []byte("clonefactoryContractPurchased(address)")
	clonefactoryContractPurchasedSigHash := crypto.Keccak256Hash(clonefactoryContractPurchasedSig)

	for {
		select {
		case <-buyer.Ctx.Done():
			contextlib.Logf(buyer.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchContractPurchase routine")
			return
		case err := <-cfSub.Err():
			contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		case cfLog := <-cfLogs:
			if cfLog.Topics[0].Hex() == clonefactoryContractPurchasedSigHash.Hex() {
				address := common.HexToAddress(cfLog.Topics[1].Hex())
				// check if contract was purchased by buyer
				hashrateContractInstance, err := implementation.NewImplementation(address, buyer.EthClient)
				if err != nil {
					contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
				}
				hashrateContractBuyer, err := hashrateContractInstance.Buyer(nil)
				if err != nil {
					contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
				}
				if hashrateContractBuyer == buyer.Account {
					contextlib.Logf(buyer.Ctx, log.LevelInfo, "Address of purchased Hashrate Contract : %s\n\n", address.Hex())

					destUrl, err := readDestUrl(buyer.EthClient, common.HexToAddress(string(address.Hex())), buyer.PrivateKey)
					if err != nil {
						contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					destMsg := msgbus.Dest{
						ID:     msgbus.DestID(msgbus.GetRandomIDString()),
						NetUrl: msgbus.DestNetUrl(destUrl),
					}
					buyer.Ps.PubWait(msgbus.DestMsg, msgbus.IDString(destMsg.ID), destMsg)

					purchasedContractValues, err := readHashrateContract(buyer.EthClient, address)
					if err != nil {
						contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					contractMsg := createContractMsg(address, purchasedContractValues, false)
					contractMsg.Dest = destMsg.ID
					contractMsg.State = msgbus.ContRunningState
					buyer.Ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contractMsg.ID), contractMsg)

					buyer.NodeOperator.Contracts[contractMsg.ID] = msgbus.ContRunningState
					buyer.Ps.SetWait(msgbus.NodeOperatorMsg, msgbus.IDString(buyer.NodeOperator.ID), buyer.NodeOperator)
				}
			}
		}
	}
}

func (buyer *BuyerContractManager) watchHashrateContract(addr msgbus.ContractID, hrLogs chan types.Log, hrSub ethereum.Subscription) {
	defer close(hrLogs)
	defer hrSub.Unsubscribe()

	// create event signatures to parse out which event was being emitted from hashrate contract
	contractClosedSig := []byte("contractClosed()")
	purchaseInfoUpdatedSig := []byte("purchaseInfoUpdated()")
	cipherTextUpdatedSig := []byte("cipherTextUpdated(string)")
	contractClosedSigHash := crypto.Keccak256Hash(contractClosedSig)
	purchaseInfoUpdatedSigHash := crypto.Keccak256Hash(purchaseInfoUpdatedSig)
	cipherTextUpdatedSigHash := crypto.Keccak256Hash(cipherTextUpdatedSig)

	// monitor events emmited by hashrate contract
	for {
		select {
		case <-buyer.Ctx.Done():
			contextlib.Logf(buyer.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchHashrateContract go routine")
			return
		case err := <-hrSub.Err():
			contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		case hLog := <-hrLogs:
			switch hLog.Topics[0].Hex() {
			case contractClosedSigHash.Hex():
				contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate Contract %s Closed \n\n", addr)

				buyer.Ps.Unpub(msgbus.ContractMsg, msgbus.IDString(addr))

				delete(buyer.NodeOperator.Contracts, addr)
				buyer.Ps.SetWait(msgbus.NodeOperatorMsg, msgbus.IDString(buyer.NodeOperator.ID), buyer.NodeOperator)

			case purchaseInfoUpdatedSigHash.Hex():
				contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate Contract %s Purchase Info Updated \n\n", addr)

				event, err := buyer.Ps.GetWait(msgbus.ContractMsg, msgbus.IDString(addr))
				if err != nil {
					contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
				}
				if event.Err != nil {
					contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
				}
				contractMsg := event.Data.(msgbus.Contract)

				updatedContractValues, err := readHashrateContract(buyer.EthClient, common.HexToAddress(string(addr)))
				if err != nil {
					contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
				}
				updateContractMsg(&contractMsg, updatedContractValues)
				buyer.Ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contractMsg.ID), contractMsg)

			case cipherTextUpdatedSigHash.Hex():
				contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate Contract %s Cipher Text Updated \n\n", addr)

				event, err := buyer.Ps.GetWait(msgbus.ContractMsg, msgbus.IDString(addr))
				if err != nil {
					contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", err)
				}
				if event.Err != nil {
					contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Purchased Contract Failed: %v", event.Err)
				}
				contractMsg := event.Data.(msgbus.Contract)
				event, err = buyer.Ps.GetWait(msgbus.DestMsg, msgbus.IDString(contractMsg.Dest))
				if err != nil {
					contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Dest Failed: %v", err)
				}
				if event.Err != nil {
					contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Dest Failed: %v", event.Err)
				}
				destMsg := event.Data.(msgbus.Dest)

				destUrl, err := readDestUrl(buyer.EthClient, common.HexToAddress(string(addr)), buyer.PrivateKey)
				if err != nil {
					contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
				}
				destMsg.NetUrl = msgbus.DestNetUrl(destUrl)
				buyer.Ps.SetWait(msgbus.DestMsg, msgbus.IDString(destMsg.ID), destMsg)
			}
		}
	}
}

func (buyer *BuyerContractManager) closeOutMonitor(minerCh msgbus.EventChan, contractCh msgbus.EventChan, contractId msgbus.ContractID) {
	for {
		select {
		case <-buyer.Ctx.Done():
			contextlib.Logf(buyer.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling closeOutMonitor go routine")
			return
		case event := <-minerCh:
			if event.EventType == msgbus.PublishEvent || event.EventType == msgbus.UpdateEvent || event.EventType == msgbus.UnpublishEvent {
				// check hashrate is being fulfilled for all running contracts
				time.Sleep(time.Second * time.Duration(buyer.TimeThreshold)) // give buffer time for total hashrate to adjust to multiple updates
				contractClosed := buyer.checkHashRate(contractId)
				if contractClosed {
					return
				}
			}
		case event := <-contractCh:
			if event.EventType == msgbus.UnpublishEvent {
				return
			}
			if event.EventType == msgbus.PublishEvent || event.EventType == msgbus.UpdateEvent {
				// check hashrate is being fulfilled after contract update
				contractClosed := buyer.checkHashRate(contractId)
				if contractClosed {
					return
				}
			}
		}
	}
}

func (buyer *BuyerContractManager) checkHashRate(contractId msgbus.ContractID) bool {
	// check for miners delivering hashrate for this contract
	totalHashrate := 0
	event, err := buyer.Ps.GetWait(msgbus.ContractMsg, msgbus.IDString(contractId))
	if err != nil {
		contextlib.Logf(buyer.Ctx, log.LevelPanic, "Getting Hashrate Contract Failed: %v", err)
	}
	contract := event.Data.(msgbus.Contract)
	miners, err := buyer.Ps.MinerGetAllWait()
	if err != nil {
		contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to get all miners, Fileline::%s, Error::", lumerinlib.FileLine()), err)
	}

	var miner *msgbus.Miner
	for i := range miners {
		miner, err = buyer.Ps.MinerGetWait(miners[i])
		if err != nil {
			contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to get miner, Fileline::%s, Error::", lumerinlib.FileLine()), err)
		}
		if _,ok := miner.Contracts[contractId]; !ok {
			totalHashrate += miner.CurrentHashRate
		}
	}

	//hashrateTolerance := float64(contract.Limit) / 100
	hashrateTolerance := float64(HASHRATE_LIMIT) / 100
	promisedHashrateMin := int(float64(contract.Speed) * (1 - hashrateTolerance))

	contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate being sent to contract %s: %d\n", contractId, totalHashrate)
	if totalHashrate <= promisedHashrateMin {
		contextlib.Logf(buyer.Ctx, log.LevelInfo, "Closing out contract %s for not meeting hashrate requirements\n", contractId)
		var wg sync.WaitGroup
		wg.Add(1)
		err := setContractCloseOut(buyer.EthClient, buyer.Account, buyer.PrivateKey, common.HexToAddress(string(contractId)), &wg, &buyer.CurrentNonce, 0, buyer.Ps, buyer.NodeOperator)
		if err != nil {
			contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Contract Close Out failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
		}
		wg.Wait()
		return true
	}

	contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate promised by contract %s is being fulfilled\n", contractId)
	return false
}

func HdWalletKeys(mnemonic string, accountIndex int) (accounts.Account, string) {
	wallet, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		panic(fmt.Sprintf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err))
	}
	path := hdwallet.MustParseDerivationPath("m/44'/60'/0'/0/" + fmt.Sprint(accountIndex))
	Account, err := wallet.Derive(path, false)
	if err != nil {
		panic(fmt.Sprintf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err))
	}
	PrivateKey, err := wallet.PrivateKeyHex(Account)
	if err != nil {
		panic(fmt.Sprintf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err))
	}

	fmt.Println("Contract Manager Account Address:", Account)

	return Account, PrivateKey
}

func setUpClient(clientAddress string, contractManagerAccount common.Address) (client *ethclient.Client, err error) {
	client, err = ethclient.Dial(clientAddress)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return client, err
	}

	fmt.Printf("Connected to rpc client at %v\n", clientAddress)

	var balance *big.Int
	balance, err = client.BalanceAt(context.Background(), contractManagerAccount, nil)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return client, err
	}
	fbalance := new(big.Float)
	fbalance.SetString(balance.String())
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))

	fmt.Println("Balance of contract manager Account:", ethValue, "ETH")

	return client, err
}

func SubscribeToContractEvents(client *ethclient.Client, contractAddress common.Address) (chan types.Log, ethereum.Subscription, error) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return logs, sub, err
	}

	return logs, sub, err
}

func readHashrateContract(client *ethclient.Client, contractAddress common.Address) (hashrateContractValues, error) {
	var contractValues hashrateContractValues

	instance, err := implementation.NewImplementation(contractAddress, client)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return contractValues, err
	}

	state, price, limit, speed, length, startingBlockTimestamp, buyer, seller, _, err := instance.GetPublicVariables(&bind.CallOpts{})
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
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

func readDestUrl(client *ethclient.Client, contractAddress common.Address, privateKeyString string) (string, error) {
	instance, err := implementation.NewImplementation(contractAddress, client)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return "", err
	}

	fmt.Printf("Getting Dest url from contract %s\n\n", contractAddress)

	encryptedDestUrl, err := instance.EncryptedPoolData(nil)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
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

func setContractCloseOut(client *ethclient.Client, fromAddress common.Address, privateKeyString string, contractAddress common.Address, wg *sync.WaitGroup, CurrentNonce *nonce, closeOutType uint, Ps *msgbus.PubSub, NodeOperator msgbus.NodeOperator) error {
	defer wg.Done()
	defer CurrentNonce.mutex.Unlock()

	CurrentNonce.mutex.Lock()

	instance, err := implementation.NewImplementation(contractAddress, client)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return err
	}

	PrivateKey, err := crypto.HexToECDSA(privateKeyString)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return err
	}

	chainId, err := client.ChainID(context.Background())
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(PrivateKey, chainId)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return err
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return err
	}
	auth.GasPrice = gasPrice
	auth.GasLimit = uint64(3000000) // in units
	auth.Value = big.NewInt(0)      // in wei

	CurrentNonce.nonce, err = client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return err
	}
	auth.Nonce = big.NewInt(int64(CurrentNonce.nonce))

	tx, err := instance.SetContractCloseOut(auth, big.NewInt(int64(closeOutType)))
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return err
	}

	fmt.Printf("tx sent: %s\n\n", tx.Hash().Hex())
	fmt.Println("Closing Out Contract: ", contractAddress)

	event, err := Ps.GetWait(msgbus.ContractMsg, msgbus.IDString(contractAddress.Hex()))
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return err
	}
	if event.Err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return err
	}
	contractMsg := event.Data.(msgbus.Contract)
	if contractMsg.State == msgbus.ContRunningState {
		contractMsg.State = msgbus.ContAvailableState
		contractMsg.Buyer = ""
		Ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contractMsg.ID), contractMsg)

		NodeOperator.Contracts[msgbus.ContractID(contractAddress.Hex())] = msgbus.ContAvailableState
		Ps.SetWait(msgbus.NodeOperatorMsg, msgbus.IDString(NodeOperator.ID), NodeOperator)
	}
	return err
}

func createContractMsg(contractAddress common.Address, contractValues hashrateContractValues, isSeller bool) msgbus.Contract {
	convertToMsgBusState := map[uint8]msgbus.ContractState{
		AvailableState: msgbus.ContAvailableState,
		RunningState:   msgbus.ContRunningState,
	}

	var contractMsg msgbus.Contract
	contractMsg.IsSeller = isSeller
	contractMsg.ID = msgbus.ContractID(contractAddress.Hex())
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
