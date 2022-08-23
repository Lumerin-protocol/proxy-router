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
	logger interfaces.ILogger
}

func (gateway *EthereumGateway) SubscribeToContractEvents(contract interfaces.ISellerContractModel) (chan interop.BlockchainEvent, interop.BlockchainEventSubscription, error) {

	ctx := context.TODO()

	query := ethereum.FilterQuery{
		Addresses: []interop.BlockchainAddress{common.HexToAddress(contract.GetAddress())},
	}

	logs := make(chan interop.BlockchainEvent)
	gateway.logger.Debugf("subscribing to blockchain client log events: %v", contract.GetId())
	sub, err := gateway.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		gateway.logger.Debugf("error subscribing to contract events: %v", err)
		// fmt.Printf("Funcname::%s, Fileline::%s, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		return logs, sub, err
	}

	contractPurchasedSig := []byte("contractPurchased(address)")
	// contractClosedSig := []byte("contractClosed()")
	// purchaseInfoUpdatedSig := []byte("purchaseInfoUpdated()")
	// cipherTextUpdatedSig := []byte("cipherTextUpdated(string)")
	contractPurchasedSigHash := crypto.Keccak256Hash(contractPurchasedSig)
	// contractClosedSigHash := crypto.Keccak256Hash(contractClosedSig)
	// purchaseInfoUpdatedSigHash := crypto.Keccak256Hash(purchaseInfoUpdatedSig)
	// cipherTextUpdatedSigHash := crypto.Keccak256Hash(cipherTextUpdatedSig)

	// routine monitoring and acting upon events emmited by hashrate contract
	go func() {
		defer close(logs)
		defer sub.Unsubscribe()
		for {
			select {
			// TODO: handle errors
			case err := <-sub.Err():
				gateway.logger.Error(err)
			//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Funcname::%s, Fileline::%s, Error::", lumerinlib.Funcname(), lumerinlib.FileLine()), err)
			case <-ctx.Done():
				gateway.logger.Infof("Cancelling current contract manager context: cancelling watchHashrateContract go routine;  contract address: %v", contract.GetAddress())
				// contextlib.Logf(seller.Ctx, log.LevelInfo, "Cancelling current contract manager context: cancelling watchHashrateContract go routine")
				return
			case hLog := <-logs:
				gateway.logger.Debugf("contract event")
				switch hLog.Topics[0].Hex() {
				case contractPurchasedSigHash.Hex():

					gateway.logger.Debugf("contract purchased event")
					destUrl, err := gateway.readDestUrl(common.HexToAddress(contract.GetAddress()), contract.GetPrivateKey())

					if err != nil {
						//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					}

					buyer := common.HexToAddress(hLog.Topics[1].Hex())
					contract.SetDestination(destUrl)
					contract.SetBuyerAddress(buyer.Hex())
					contract.Execute()
					// contractService.Run(destUrl, buyer.Hex(), address)
					// case cipherTextUpdatedSigHash.Hex():

					// 	destUrl, err := readDestUrl(seller.EthClient, common.HexToAddress(string(addr)), seller.PrivateKey)

					// 	if err != nil {
					// 		//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading dest url failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					// 	}

					// 	hashrateContractMsg.Dest = destUrl

					// 	seller.Ps.HandleDestinationUpdated(hashrateContractMsg)

					// case contractClosedSigHash.Hex():
					// 	seller.Ps.HandleContractClosed(hashrateContractMsg)

					// case purchaseInfoUpdatedSigHash.Hex():
					// 	updatedContractValues, err := readHashrateContract(seller.EthClient, common.HexToAddress(string(addr)))

					// 	if err != nil {
					// 		//contextlib.Logf(seller.Ctx, log.LevelPanic, fmt.Sprintf("Reading hashrate contract failed, Fileline::%s, Error::", lumerinlib.FileLine()), err)
					// 	}

					// 	updateContractMsg(hashrateContractMsg, updatedContractValues)

					// 	seller.Ps.HandleContractUpdated(hashrateContractMsg)

					// }
				}
			}
		}
	}()

	return logs, sub, err
}

func (gateway *EthereumGateway) readDestUrl(contractAddress interop.BlockchainAddress, privateKeyString string) (string, error) {

	client := gateway.client
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

func (gateway *EthereumGateway) SetContractCloseOut(contract interfaces.ISellerContractModel) error {
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
