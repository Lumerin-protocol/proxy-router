package contractmanager

import (
	"fmt"
	"log"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var sellerAddress = common.HexToAddress("0xa1bA6996176dDd0D0107dC7e3830e9cf8c1bbe96")
var sellerPrivateKey = "afb649709eacc32166c9c5e4935c58efeeed89a0af056a5cae3da266413e5217"
var buyerAddress = common.HexToAddress("0x0FDcC9fF7D6F5f79c4e80e797916713a2d05A9cA")
var buyerPrivateKey = "5a25d76802639b7df2f8b9c0339e67662db8a5e81368288dde5bcef0bf606de9"
var gethNodeAddress = "wss://ropsten.infura.io/ws/v3/4b68229d56fe496e899f07c3d41cb08a"

var clonefactoryAddress common.Address = common.HexToAddress("0xe91be01493f4ae28297790277303926aaec604dc")

var hashrateContractAddress common.Address = common.HexToAddress("0xecA5878daB5Bd7a4052Dc5BE3cd993a1507AA040") // 0x597e311EEB16a4d213389F1661272B26BDE0E698 0x7E4f2cea58705482dBE8F1269996b5120db321a2
var poolUrl = "stratum+tcp://rbajollari.contract1:@btc.f2pool.com:3333"

func TestHashrateContractCreation(t *testing.T) {
	// hashrate contract params
	price := 0
	limit := 20
	speed := 50000000000000
	length := 7200

	client, err := setUpClient(gethNodeAddress, sellerAddress)
	if err != nil {
		log.Fatalf("Error::%v", err)
	}

	CreateHashrateContract(client, sellerAddress, sellerPrivateKey, clonefactoryAddress, price, limit, speed, length, clonefactoryAddress)

	// subcribe to creation events emitted by clonefactory contract
	cfLogs, cfSub, _ := SubscribeToContractEvents(client, clonefactoryAddress)
	// create event signature to parse out creation event
	contractCreatedSig := []byte("contractCreated(address,string)")
	contractCreatedSigHash := crypto.Keccak256Hash(contractCreatedSig)
	for {
		select {
		case err := <-cfSub.Err():
			log.Fatalf("Error::%v", err)
		case cfLog := <-cfLogs:

			if cfLog.Topics[0].Hex() == contractCreatedSigHash.Hex() {
				hashrateContractAddress := common.HexToAddress(cfLog.Topics[1].Hex())
				fmt.Printf("Address of created Hashrate Contract: %v\n\n", hashrateContractAddress.Hex())
			}
		}
	}
}

func TestHashrateContractPurchase(t *testing.T) {

	client, err := setUpClient(gethNodeAddress, buyerAddress)
	if err != nil {
		log.Fatalf("Error::%v", err)
	}

	PurchaseHashrateContract(client, buyerAddress, buyerPrivateKey, clonefactoryAddress, hashrateContractAddress, buyerAddress, poolUrl)

	// subcribe to purchase events emitted by clonefactory contract
	cfLogs, cfSub, _ := SubscribeToContractEvents(client, clonefactoryAddress)
	// create event signature to parse out purchase event
	clonefactoryContractPurchasedSig := []byte("clonefactoryContractPurchased(address)")
	clonefactoryContractPurchasedSigHash := crypto.Keccak256Hash(clonefactoryContractPurchasedSig)
	for {
		select {
		case err := <-cfSub.Err():
			log.Fatalf("Error::%v", err)
		case cfLog := <-cfLogs:

			if cfLog.Topics[0].Hex() == clonefactoryContractPurchasedSigHash.Hex() {
				hashrateContractAddress := common.HexToAddress(cfLog.Topics[1].Hex())
				fmt.Printf("Address of purchased Hashrate Contract: %v\n\n", hashrateContractAddress.Hex())
			}
		}
	}
}
