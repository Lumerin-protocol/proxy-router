package contractmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
	"gitlab.com/TitanInd/hashrouter/lumerinlib/clonefactory"
	"gitlab.com/TitanInd/hashrouter/lumerinlib/implementation"
)

type BuyerContractManager struct {
	ContractFactory     interfaces.IContractFactory
	Ps                  interfaces.IContractsService
	Logger              interfaces.ILogger
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

func (buyer *BuyerContractManager) Run(ctx context.Context) (err error) {
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
		// buyer.Ps.OnContractCreated(func(newContract interfaces.IContractModel) {
		// 	buyer.Logger.Infof("created a contract %v", newContract.GetId())
		// 	addr := common.HexToAddress(string(newContract.GetAddress()))
		// 	hrLogs, hrSub, err := SubscribeToContractEvents(buyer.EthClient, addr)
		// 	if err != nil {
		// 		//contextlib.Logf(buyer.Ctx, log.LevelPanic, fmt.Sprintf("Failed to subscribe to events on hashrate contract %s, Fileline::%s, Error::", addr, lumerinlib.FileLine()), err)
		// 	}
		// 	go buyer.WatchHashrateContract(string(addr.Hex()), hrLogs, hrSub)

		// 	go buyer.closeOutMonitor(newContract.GetAddress())
		// })
	}()
	return nil
}

func (buyer *BuyerContractManager) SetupExistingContracts() (err error) {
	var contractValues []hashrateContractValues
	var contractMsgs []interfaces.ISellerContractModel
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
					fmt.Printf(destUrl)
					// contractMsg.SetDestination(destUrl)

					buyer.Ps.HandleContractPurchased(destUrl, contractMsg.GetId(), address.Hex())

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
			contract, err := buyer.Ps.GetContract(addr)

			if err == nil {
				buyer.Logger.Errorf(err.Error())
			}

			switch hLog.Topics[0].Hex() {
			case contractClosedSigHash.Hex():

				buyer.Ps.HandleContractClosed(contract)

			case purchaseInfoUpdatedSigHash.Hex():
				// buyer.Ps.HandleContractUpdated(contract.GetPriceRequirement(), contract.GetTimeRequirement(), contract.GetHashrateRequirement())

			case cipherTextUpdatedSigHash.Hex():
				// buyer.Ps.HandleDestinationUpdated(contract.GetDestination())
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
