package contractmanager

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"

	//"encoding/hex"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	//"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/ethclient"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"

	// "gitlab.com/TitanInd/hashrouter/cmd/log"
	// "gitlab.com/TitanInd/hashrouter/cmd/msgbus"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
	"gitlab.com/TitanInd/hashrouter/lumerinlib"

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
// 	return false
// }

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

func createContractMsg(contractAddress interop.BlockchainAddress, contractValues hashrateContractValues, isSeller bool) interfaces.ISellerContractModel {

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
