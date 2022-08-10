package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
	"gitlab.com/TitanInd/hashrouter/lumerinlib"
	"gitlab.com/TitanInd/hashrouter/lumerinlib/implementation"
)

type EthereumGateway struct {
	interfaces.IBlockchainGateway
	client *interop.BlockchainClient
	wallet *EthereumWallet
}

func (gateway *EthereumGateway) SubscribeToContractEvents(address string) (chan interop.BlockchainEvent, interop.BlockchainEventSubscription, error) {
	query := ethereum.FilterQuery{
		Addresses: []interop.BlockchainAddress{common.HexToAddress(address)},
	}

	logs := make(chan interop.BlockchainEvent)
	sub, err := gateway.client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		// fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return logs, sub, err
	}

	return logs, sub, err
}

func (gateway *EthereumGateway) SetContractCloseOut(contract interfaces.IContractModel) error {
	client := gateway.client
	privateKeyString := contract.GetPrivateKey()
	fromAddress := contract.GetBuyerAddress()
	contractAddress := contract.GetAddress() //, wg *sync.WaitGroup, CurrentNonce *nonce, closeOutType uint, NodeOperator *NodeOperator
	currentNonce := contract.GetCurrentNonce()
	var wg sync.WaitGroup
	defer wg.Done()
	wg.Add(1)
	// defer CurrentNonce.mutex.Unlock()

	// CurrentNonce.mutex.Lock()

	instance, err := implementation.NewImplementation(gateway.wallet.HexToAddress(contractAddress), client)

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

	currentNonce, err = client.PendingNonceAt(context.Background(), gateway.wallet.HexToAddress(fromAddress))
	if err != nil {
		fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return err
	}

	auth.Nonce = big.NewInt(int64(currentNonce))

	tx, err := instance.SetContractCloseOut(auth, big.NewInt(int64(contract.GetCloseOutType())))
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

func NewBlockchainGateway(ethClient *interop.BlockchainClient) (client interfaces.IBlockchainGateway, err error) {

	return &EthereumGateway{
		client: ethClient,
	}, err
}
