package contractmanager

// import (
// 	"context"
// 	"sync"

// 	"github.com/ethereum/go-ethereum/accounts"
// 	"github.com/ethereum/go-ethereum/accounts/abi/bind"
// 	"github.com/ethereum/go-ethereum/common"
// 	"github.com/ethereum/go-ethereum/ethclient"
// 	"gitlab.com/TitanInd/hashrouter/config"
// 	"gitlab.com/TitanInd/hashrouter/constants"
// 	"gitlab.com/TitanInd/hashrouter/interfaces"
// 	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
// 	"gitlab.com/TitanInd/lumerin/lumerinlib/implementation"
// )

// const (
// 	AvailableState uint8 = 0
// 	RunningState   uint8 = 1
// 	HASHRATE_LIMIT       = 20
// )

// type ContractModel struct {
// 	Ps                    interfaces.IEventManager
// 	RoutableStreamService interfaces.IRoutableStreamService
// 	ContractGateway       interfaces.IContractGateway
// 	EthClient             *ethclient.Client
// 	CloneFactoryAddress   common.Address
// 	Account               common.Address
// 	PrivateKey            string
// 	CurrentNonce          nonce
// 	NodeOperator          *NodeOperator
// 	Ctx                   context.Context
// 	ClaimFunds            bool
// 	Logger                interfaces.ILogger
// 	//Buyer only
// 	TimeThreshold int
// }

// func (model *ContractModel) ReadHashrateContract(contractAddress common.Address) (hashrateContractValues, error) {
// 	var contractValues hashrateContractValues

// 	instance, err := implementation.NewImplementation(contractAddress, model.EthClient)

// 	if err != nil {
// 		model.Logger.Error(err)
// 		return contractValues, err
// 	}

// 	state, price, limit, speed, length, startingBlockTimestamp, buyer, seller, _, err := instance.GetPublicVariables(&bind.CallOpts{})
// 	if err != nil {
// 		model.Logger.Error(err)
// 		return contractValues, err
// 	}
// 	contractValues.State = state
// 	contractValues.Price = int(price.Int64())
// 	contractValues.Limit = int(limit.Int64())
// 	contractValues.Speed = int(speed.Int64())
// 	contractValues.Length = int(length.Int64())
// 	contractValues.StartingBlockTimestamp = int(startingBlockTimestamp.Int64())
// 	contractValues.Buyer = buyer
// 	contractValues.Seller = seller

// 	return contractValues, err
// }
// func createContractMsg(contractAddress common.Address, contractValues hashrateContractValues, isSeller bool) Contract {
// 	convertToMsgBusState := map[uint8]string{
// 		AvailableState: constants.ContAvailableState,
// 		RunningState:   constants.ContRunningState,
// 	}

// 	var contractMsg Contract
// 	contractMsg.IsSeller = isSeller
// 	contractMsg.ID = string(contractAddress.Hex())
// 	contractMsg.State = convertToMsgBusState[contractValues.State]
// 	contractMsg.Buyer = string(contractValues.Buyer.Hex())
// 	contractMsg.Price = contractValues.Price
// 	contractMsg.Limit = contractValues.Limit
// 	contractMsg.Speed = contractValues.Speed
// 	contractMsg.Length = contractValues.Length
// 	contractMsg.StartingBlockTimestamp = contractValues.StartingBlockTimestamp

// 	return contractMsg
// }

// func updateContractMsg(contractMsg *msgbus.Contract, contractValues hashrateContractValues) {
// 	contractMsg.Price = contractValues.Price
// 	contractMsg.Limit = contractValues.Limit
// 	contractMsg.Speed = contractValues.Speed
// 	contractMsg.Length = contractValues.Length
// }

// type NodeOperator struct {
// 	ID                     string
// 	IsBuyer                bool
// 	DefaultDest            string
// 	EthereumAccount        string
// 	TotalAvailableHashRate int
// 	UnusedHashRate         int
// 	Contracts              map[string]string
// }

// type Contract struct {
// 	IsSeller               bool
// 	ID                     string
// 	State                  string
// 	Buyer                  string
// 	Price                  int
// 	Limit                  int
// 	Speed                  int
// 	Length                 int
// 	StartingBlockTimestamp int
// 	Dest                   string
// }

// type hashrateContractValues struct {
// 	State                  uint8
// 	Price                  int
// 	Limit                  int
// 	Speed                  int
// 	Length                 int
// 	StartingBlockTimestamp int
// 	Buyer                  common.Address
// 	Seller                 common.Address
// }

// type nonce struct {
// 	mutex sync.Mutex
// 	nonce uint64
// }

// func NewContractModel(ctx context.Context, logger interfaces.ILogger, configuration *config.Config, eventManager interfaces.IEventManager, client *ethclient.Client, nodeOperator *NodeOperator, account *accounts.Account, privateKey string) *ContractModel {

// 	// var client *ethclient.Client
// 	// client, err = setUpClient(ethNodeAddr, seller.Account)
// 	// if err != nil {
// 	// 	return err
// 	// }
// 	contractConfig := configuration.Contract
// 	return &ContractModel{
// 		Ctx:        ctx,
// 		Ps:         eventManager,
// 		EthClient:  client,
// 		ClaimFunds: contractConfig.ClaimFunds,
// 		// ethNodeAddr:  contractConfig.EthNodeAddr,
// 		// mnemonic:     contractConfig.Mnemonic,
// 		// accountIndex: contractConfig.AccountIndex,
// 		Account:    account.Address,
// 		PrivateKey: privateKey,

// 		CloneFactoryAddress: common.HexToAddress(contractConfig.CloneFactoryAddress),

// 		NodeOperator: nodeOperator,
// 		Logger:       logger,
// 		// seller.NodeOperator.EthereumAccount = seller.Account.Hex(),
// 	}
// }
