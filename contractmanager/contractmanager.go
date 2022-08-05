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
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	//"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/ethclient"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"

	// "gitlab.com/TitanInd/hashrouter/cmd/log"
	// "gitlab.com/TitanInd/hashrouter/cmd/msgbus"
	"gitlab.com/TitanInd/hashrouter/config"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
	"gitlab.com/TitanInd/hashrouter/lumerinlib"
	"gitlab.com/TitanInd/hashrouter/lumerinlib/clonefactory"

	// contextlib "gitlab.com/TitanInd/hashrouter/lumerinlib/context"
	"gitlab.com/TitanInd/hashrouter/lumerinlib/implementation"
)

const (
	AvailableState uint8 = 0
	RunningState   uint8 = 1
	HASHRATE_LIMIT       = 20
)

const (
	ContAvailableState string = "AvailableState"
	ContRunningState   string = "RunningState"
)

var ContractStateEnum = map[uint8]string{
	AvailableState: ContAvailableState,
	RunningState:   ContRunningState,
}

const (
	NoEvent           string = "NoEvent"
	UpdateEvent       string = "UpdEvent"
	DeleteEvent       string = "DelEvent"
	GetEvent          string = "GetEvent"
	GetIndexEvent     string = "GetIdxEvent"
	SearchEvent       string = "SearchEvent"
	SearchIndexEvent  string = "SearchIndexEvent"
	PublishEvent      string = "PubEvent"
	UnpublishEvent    string = "UnpubEvent"
	SubscribedEvent   string = "SubEvent"
	UnsubscribedEvent string = "UnsubEvent"
	RemovedEvent      string = "RemovedEvent"
)

const (
	NoMsg                    string = "NoMsg"
	ConfigMsg                string = "ConfigMsg"
	ContractManagerConfigMsg string = "ContractManagerConfigMsg"
	DestMsg                  string = "DestMsg"
	NodeOperatorMsg          string = "NodeOperatorMsg"
	ContractMsg              string = "ContractMsg"
	MinerMsg                 string = "MinerMsg"
	ConnectionMsg            string = "ConnectionMsg"
	LogMsg                   string = "LogMsg"
	ValidateMsg              string = "ValidateMsg"
)

type hashrateContractValues struct {
	State                  uint8
	Price                  int
	Limit                  int
	Speed                  int
	Length                 int
	StartingBlockTimestamp int
	Buyer                  interop.BlockchainAddress
	Seller                 interop.BlockchainAddress
}

type Dest struct {
	ID     string
	NetUrl string
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

type ContractManagerConfig struct {
	Mnemonic            string
	AccountIndex        int
	EthNodeAddr         string
	ClaimFunds          bool
	TimeThreshold       int
	CloneFactoryAddress string
	LumerinTokenAddress string
	ValidatorAddress    string
	ProxyAddress        string
}

type nonce struct {
	mutex sync.Mutex
	nonce uint64
}

type Contract struct {
	interfaces.IContractManager
	ethereumGateway        interfaces.IBlockchainGateway
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

func (c *Contract) HasDestination() bool {
	return c.Dest != ""
}

func (c *Contract) SetDestination(destination string) {
	c.Dest = destination
}

func (c *Contract) IsAvailable() bool {
	return c.State == ContAvailableState
}

func (c *Contract) SubscribeToContractEvents(address string) (chan interop.BlockchainEvent, interop.BlockchainEventSubscription, error) {
	return c.ethereumGateway.SubscribeToContractEvents(address)
}

func (c *Contract) GetHexAddress() string {
	return c.ID
}

var _ interfaces.IContractModel = (*Contract)(nil)

type SellerContractManager struct {
	ContractFactory     interfaces.IContractFactory
	Ps                  interfaces.IContractsService
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

type BuyerContractManager struct {
	ContractFactory     interfaces.IContractFactory
	Ps                  interfaces.IContractsService
	EthClient           *ethclient.Client
	CloneFactoryAddress interop.BlockchainAddress
	Account             interop.BlockchainAddress
	PrivateKey          string
	CurrentNonce        nonce
	TimeThreshold       int
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
	account *interop.BlockchainAccount,
	privateKey string) interfaces.ContractManager {

	if configuration.Contract.IsBuyer {
		buyer := &BuyerContractManager{
			ContractFactory:     contractFactory,
			Ctx:                 ctx,
			Ps:                  contractsService,
			TimeThreshold:       configuration.Contract.TimeThreshold,
			EthClient:           client,
			CloneFactoryAddress: common.HexToAddress(configuration.Contract.CloneFactoryAddress),
			NodeOperator:        nodeOperator,
			PrivateKey:          privateKey,
			Account:             account.Address,
		}

		if buyer.NodeOperator.Contracts == nil {
			buyer.NodeOperator.Contracts = make(map[string]string)
		}

		return buyer

	}

	seller := &SellerContractManager{
		ContractFactory:     contractFactory,
		Ctx:                 ctx,
		Ps:                  contractsService,
		EthClient:           client,
		CloneFactoryAddress: common.HexToAddress(configuration.Contract.CloneFactoryAddress),
		NodeOperator:        nodeOperator,
		ClaimFunds:          configuration.Contract.ClaimFunds,
		PrivateKey:          privateKey,
		Account:             account.Address,
	}

	// ethNodeAddr := configuration.Contract.EthNodeAddr
	mnemonic := configuration.Contract.Mnemonic
	accountIndex := configuration.Contract.AccountIndex

	Account, PrivateKey := HdWalletKeys(mnemonic, accountIndex)
	seller.Account = Account.Address
	seller.PrivateKey = PrivateKey

	seller.EthClient = client
	seller.CloneFactoryAddress = common.HexToAddress(configuration.Contract.CloneFactoryAddress)

	seller.NodeOperator = nodeOperator
	seller.NodeOperator.EthereumAccount = seller.Account.Hex()

	return seller
	// ethNodeAddr :=  configuration.Contract.EthNodeAddr
	// mnemonic := configuration.Contract.Mnemonic
	// accountIndex := configuration.Contract.AccountIndex

	// Account, PrivateKey := HdWalletKeys(mnemonic, accountIndex)

	// var client *ethclient.Client
	// client, err = setUpClient(ethNodeAddr, buyer.Account)
	// if err != nil {
	// 	return err
	// }
}

func (seller *SellerContractManager) Start() (err error) {
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

			if newContract.IsAvailable() {
				addr := common.HexToAddress(newContract.GetHexAddress())
				hrLogs, hrSub, err := SubscribeToContractEvents(seller.EthClient, addr)
				if err != nil {
					//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to events on hashrate contract %s, Fileline::%s, Error::", newContract.ID, lumerinlib.FileLine()), err)
				}
				go seller.WatchHashrateContract(addr.Hex(), hrLogs, hrSub)
			}
		})
	}()
	fmt.Printf("Error in start: %v\n", err)

	return err
}

func (seller *SellerContractManager) SetupExistingContracts() (err error) {
	var contractValues []hashrateContractValues
	var contractMsgs []interfaces.IContractModel

	sellerContracts, err := seller.ReadContracts()
	if err != nil {
		return err
	}
	//contextlib.Logf(seller.Ctx, log.LevelInfo, "Existing Seller Contracts: %v", sellerContracts)

	// get existing dests in msgbus to see if contract's dest already exists
	existingDests := seller.Ps.GetDestinations()
	for i := range sellerContracts {
		id := string(sellerContracts[i].Hex())
		if _, ok := seller.NodeOperator.Contracts[id]; !ok {
			contract, err := readHashrateContract(seller.EthClient, sellerContracts[i])
			if err != nil {
				return err
			}
			contractValues = append(contractValues, contract)
			contractMsgs = append(contractMsgs, createContractMsg(sellerContracts[i], contractValues[i], true))

			seller.NodeOperator.Contracts[string(sellerContracts[i].Hex())] = ContAvailableState

			if contractValues[i].State == RunningState {
				seller.NodeOperator.Contracts[string(sellerContracts[i].Hex())] = ContRunningState

				destUrl, err := readDestUrl(seller.EthClient, sellerContracts[i], seller.PrivateKey)
				if err != nil {
					//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
				}

				// if msgbus has dest with same target address, use that as contract msg dest
				for _, existingDest := range existingDests {

					if existingDest == destUrl {
						contractMsgs[i].SetDestination(existingDest)
					}
				}

				// msgbus does not have dest with that target address
				if !contractMsgs[i].HasDestination() {
					seller.Ps.CreateDestination(destUrl)

					contractMsgs[i].SetDestination(destUrl)
				}
			}

		}
	}

	seller.Ps.SaveContracts(contractMsgs)
	// seller.Ps.SaveContracts(contractMsgs)
	// seller.Ps.SetWait(NodeOperatorMsg, string(seller.NodeOperator.ID), seller.NodeOperator)

	return err
}

func (seller *SellerContractManager) ReadContracts() ([]interop.BlockchainAddress, error) {
	var sellerContractAddresses []interop.BlockchainAddress
	var hashrateContractInstance *implementation.Implementation
	var hashrateContractSeller interop.BlockchainAddress

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
		// case err := <-cfSub.Err():
		// contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		case <-seller.Ctx.Done():
			//contextlib.Logf(seller.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchContractCreation go routine")
			return
		case cfLog := <-cfLogs:
			if cfLog.Topics[0].Hex() == contractCreatedSigHash.Hex() {

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

					hashrateContractMsg.Dest = destUrl

					seller.Ps.HandleContractPurchased(hashrateContractMsg)

				case cipherTextUpdatedSigHash.Hex():

					destUrl, err := readDestUrl(seller.EthClient, common.HexToAddress(string(addr)), seller.PrivateKey)

					if err != nil {
						//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}

					hashrateContractMsg.Dest = destUrl

					seller.Ps.HandleContractUpdated(hashrateContractMsg)

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

func (seller *SellerContractManager) closeOutMonitor(contractMsg Contract) {
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

func (buyer *BuyerContractManager) Start() (err error) {
	err = buyer.SetupExistingContracts()
	if err != nil {
		return err
	}

	// routine for listensing to contract purchase events to update buyer with new contracts they purchased
	cfLogs, cfSub, err := SubscribeToContractEvents(buyer.EthClient, buyer.CloneFactoryAddress)
	if err != nil {
		return err
	}
	go buyer.watchContractPurchase(cfLogs, cfSub)

	// routine starts routines for buyers's contracts that monitors contract running and close events
	go func() {
		// start watch hashrate contract for existing running contracts
		for addr := range buyer.NodeOperator.Contracts {
			hrLogs, hrSub, err := SubscribeToContractEvents(buyer.EthClient, common.HexToAddress(string(addr)))
			if err != nil {
				//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to events on hashrate contract %s, Fileline::%s, Error::", addr, lumerinlib.FileLine()), err)
			}
			go buyer.WatchHashrateContract(addr, hrLogs, hrSub)

			go buyer.closeOutMonitor(addr)
		}

		// monitor new contracts getting purchased and start watch hashrate conrtract routine when they are purchased
		buyer.Ps.OnContractCreated(func(newContract interfaces.IContractModel) {
			addr := common.HexToAddress(string(newContract.GetHexAddress()))
			hrLogs, hrSub, err := SubscribeToContractEvents(buyer.EthClient, addr)
			if err != nil {
				//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to events on hashrate contract %s, Fileline::%s, Error::", addr, lumerinlib.FileLine()), err)
			}
			go buyer.WatchHashrateContract(string(addr.Hex()), hrLogs, hrSub)

			go buyer.closeOutMonitor(newContract.GetHexAddress())
		})
	}()
	return nil
}

func (buyer *BuyerContractManager) SetupExistingContracts() (err error) {
	var contractValues []hashrateContractValues
	var contractMsgs []interfaces.IContractModel
	var nodeOperatorUpdated bool

	buyerContracts, err := buyer.ReadContracts()
	if err != nil {
		return err
	}
	//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Existing Buyer Contracts: %v", buyerContracts)

	for i := range buyerContracts {
		id := string(buyerContracts[i].Hex())
		if _, ok := buyer.NodeOperator.Contracts[id]; !ok {
			contract, err := readHashrateContract(buyer.EthClient, buyerContracts[i])
			if err != nil {
				return err
			}
			contractValues = append(contractValues, contract)
			contractMsgs = append(contractMsgs, createContractMsg(buyerContracts[i], contractValues[i], false))
			//TODO: publish contract
			// buyer.Ps.PubWait(msgbus.ContractMsg, string(contractMsgs[i].ID), contractMsgs[i])

			buyer.NodeOperator.Contracts[string(buyerContracts[i].Hex())] = ContRunningState
			nodeOperatorUpdated = true
		}
	}

	if nodeOperatorUpdated {
		//TODO: publish node operator
		// buyer.Ps.PubWait(msgbus.NodeOperatorMsg, string(buyer.NodeOperator.ID), buyer.NodeOperator)
	}

	return err
}

func (buyer *BuyerContractManager) ReadContracts() ([]interop.BlockchainAddress, error) {
	var buyerContractAddresses []interop.BlockchainAddress
	var hashrateContractInstance *implementation.Implementation
	var hashrateContractBuyer interop.BlockchainAddress

	instance, err := clonefactory.NewClonefactory(buyer.CloneFactoryAddress, buyer.EthClient)
	if err != nil {
		//contextlib.Logf(buyer.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		return buyerContractAddresses, err
	}

	hashrateContractAddresses, err := instance.GetContractList(&bind.CallOpts{})
	if err != nil {
		//contextlib.Logf(buyer.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		return buyerContractAddresses, err
	}

	// parse existing hashrate contracts for ones that belong to buyer
	for i := range hashrateContractAddresses {
		hashrateContractInstance, err = implementation.NewImplementation(hashrateContractAddresses[i], buyer.EthClient)
		if err != nil {
			//contextlib.Logf(buyer.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
			return buyerContractAddresses, err
		}
		hashrateContractBuyer, err = hashrateContractInstance.Buyer(nil)
		if err != nil {
			//contextlib.Logf(buyer.Ctx, log.LevelError, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
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
			//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchContractPurchase routine")
			return
			//TODO: error handling
		// case err := <-cfSub.Err():
		//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		case cfLog := <-cfLogs:
			if cfLog.Topics[0].Hex() == clonefactoryContractPurchasedSigHash.Hex() {
				address := common.HexToAddress(cfLog.Topics[1].Hex())
				// check if contract was purchased by buyer
				hashrateContractInstance, err := implementation.NewImplementation(address, buyer.EthClient)

				if err != nil {
					//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
				}
				hashrateContractBuyer, err := hashrateContractInstance.Buyer(nil)
				if err != nil {
					//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
				}
				if hashrateContractBuyer == buyer.Account {
					//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Address of purchased Hashrate Contract : %s\n\n", address.Hex())

					destUrl, err := readDestUrl(buyer.EthClient, common.HexToAddress(string(address.Hex())), buyer.PrivateKey)
					if err != nil {
						//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}
					purchasedContractValues, err := readHashrateContract(buyer.EthClient, address)

					if err != nil {
						//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}

					contractMsg := createContractMsg(address, purchasedContractValues, false)
					contractMsg.SetDestination(destUrl)

					buyer.Ps.HandleBuyerContractPurchased(contractMsg)

				}
			}
		}
	}
}

func (buyer *BuyerContractManager) WatchHashrateContract(addr string, hrLogs chan types.Log, hrSub ethereum.Subscription) {
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
			//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchHashrateContract go routine")
			return
			//TODO: error handling
		// case err := <-hrSub.Err():
		//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
		case hLog := <-hrLogs:
			switch hLog.Topics[0].Hex() {
			case contractClosedSigHash.Hex():

				buyer.Ps.HandleBuyerContractClosed(buyer, addr)

			case purchaseInfoUpdatedSigHash.Hex():
				buyer.Ps.HandleBuyerContractUpdated(buyer, addr)

			case cipherTextUpdatedSigHash.Hex():
				buyer.Ps.HandleBuyerDestinationUpdated(buyer, addr)
			}
		}
	}
}

func (buyer *BuyerContractManager) closeOutMonitor(contractId string) {
	go func() {
		for {
			time.Sleep(time.Second * time.Duration(buyer.TimeThreshold))
			// check contract is still running
			contractClosed := buyer.Ps.CheckHashRate(contractId)
			if contractClosed {
				return
			}
		}
	}()

	for {
		select {
		case <-buyer.Ctx.Done():
			//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling closeOutMonitor go routine")
			return
		}
	}
}

// func (buyer *BuyerContractManager) checkHashRate(contractId string) bool {
// 	// check for miners delivering hashrate for this contract
// 	totalHashrate := 0
// 	contractResult, err := buyer.Ps.GetContract(contractId)
// 	if err != nil {
// 		//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to get all miners, Fileline::%s, Error::", lumerinlib.FileLine()), err)
// 	}

// 	contract := contractResult

// 	// hashrate, err := buyer.Ps.GetHashrate()
// 	// TODO: create source/contract relationship
// 	// miners, err := buyer.Ps.GetMiners()

// 	// if err != nil {
// 	// 	//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to get all miners, Fileline::%s, Error::", lumerinlib.FileLine()), err)
// 	// }

// 	// for _, miner := range miners {
// 	// 	if err != nil {
// 	// 		//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to get miner, Fileline::%s, Error::", lumerinlib.FileLine()), err)
// 	// 	}
// 	// 	if _, ok := miner.Contracts[contractId]; ok {
// 	// 		totalHashrate += int(float64(miner.CurrentHashRate) * miner.Contracts[contractId])
// 	// 	}
// 	// }

// 	hashrateTolerance := float64(HASHRATE_LIMIT) / 100
// 	promisedHashrateMin := int(float64(contract.Speed) * (1 - hashrateTolerance))

// 	//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate being sent to contract %s: %d\n", contractId, totalHashrate)
// 	if totalHashrate <= promisedHashrateMin {
// 		//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Closing out contract %s for not meeting hashrate requirements\n", contractId)
// 		var wg sync.WaitGroup
// 		wg.Add(1)
// 		err := setContractCloseOut(buyer.EthClient, buyer.Account, buyer.PrivateKey, common.HexToAddress(string(contractId)), &wg, &buyer.CurrentNonce, 0, buyer.NodeOperator)
// 		if err != nil {
// 			//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Contract Close Out failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
// 		}
// 		wg.Wait()
// 		return true
// 	}

	//contextlib.Logf(buyer.Ctx, log.LevelInfo, "Hashrate promised by contract %s is being fulfilled\n", contractId)
	return false
}

func HdWalletKeys(mnemonic string, accountIndex int) (interop.BlockchainAccount, string) {
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

func setUpClient(clientAddress string, contractManagerAccount interop.BlockchainAddress) (client *ethclient.Client, err error) {
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

func SubscribeToContractEvents(client *ethclient.Client, contractAddress interop.BlockchainAddress) (chan types.Log, ethereum.Subscription, error) {
	query := ethereum.FilterQuery{
		Addresses: []interop.BlockchainAddress{contractAddress},
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return logs, sub, err
	}

	return logs, sub, err
}

func readHashrateContract(client *ethclient.Client, contractAddress interop.BlockchainAddress) (hashrateContractValues, error) {
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

func readDestUrl(client *ethclient.Client, contractAddress interop.BlockchainAddress, privateKeyString string) (string, error) {
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

func setContractCloseOut(client *ethclient.Client, fromAddress interop.BlockchainAddress, privateKeyString string, contractAddress interop.BlockchainAddress, wg *sync.WaitGroup, CurrentNonce *nonce, closeOutType uint, NodeOperator *NodeOperator) error {
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

	//TODO: replace GetWait and SetWait
	// event, err := Ps.GetWait(ContractMsg, string(contractAddress.Hex()))
	// if err != nil {
	// 	fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	// 	return err
	// }
	// if event.Err != nil {
	// 	fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	// 	return err
	// }
	// contractMsg := event.Data.(Contract)
	// if contractMsg.State == ContRunningState {
	// 	contractMsg.State = ContAvailableState
	// 	contractMsg.Buyer = ""
	// 	Ps.SetWait(msgbus.ContractMsg, string(contractMsg.ID), contractMsg)

	// 	NodeOperator.Contracts[string(contractAddress.Hex())] = ContAvailableState
	// 	Ps.SetWait(msgbus.NodeOperatorMsg, string(NodeOperator.ID), NodeOperator)
	// }
	return err
}

func createContractMsg(contractAddress interop.BlockchainAddress, contractValues hashrateContractValues, isSeller bool) interfaces.IContractModel {

	var contractMsg *Contract
	contractMsg.IsSeller = isSeller
	contractMsg.ID = string(contractAddress.Hex())
	contractMsg.State = ContractStateEnum[contractValues.State]
	contractMsg.Buyer = string(contractValues.Buyer.Hex())
	contractMsg.Price = contractValues.Price
	contractMsg.Limit = contractValues.Limit
	contractMsg.Speed = contractValues.Speed
	contractMsg.Length = contractValues.Length
	contractMsg.StartingBlockTimestamp = contractValues.StartingBlockTimestamp

	return contractMsg
}

func updateContractMsg(contractMsg *Contract, contractValues hashrateContractValues) {
	contractMsg.Price = contractValues.Price
	contractMsg.Limit = contractValues.Limit
	contractMsg.Speed = contractValues.Speed
	contractMsg.Length = contractValues.Length
}
