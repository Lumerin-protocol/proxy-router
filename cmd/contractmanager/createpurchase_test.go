package contractmanager

import (
	"fmt"
	"log"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var sellerAddress = common.HexToAddress("0xC3AcdAE18291bFEB0671d1caAb1d13Fe04164f75")
var sellerPrivateKey = "24c1613e07ac889f90c08f728f281e61f5e6b95d3d69c307513599f463c7d237"
var buyerAddress = common.HexToAddress("0x0FDcC9fF7D6F5f79c4e80e797916713a2d05A9cA")
var buyerPrivateKey = "5a25d76802639b7df2f8b9c0339e67662db8a5e81368288dde5bcef0bf606de9"
var gethNodeAddress = "wss://ropsten.infura.io/ws/v3/4b68229d56fe496e899f07c3d41cb08a"

var clonefactoryAddress common.Address = common.HexToAddress("0xe91be01493f4ae28297790277303926aaec604dc")

var hashrateContractAddress common.Address = common.HexToAddress("0x3b6fE2c6AcD5B52a703a9653f4af44B1176978f4") 
// 0x4b6cc541CB35F21323077a84EDE6A662155a0A83 0x4b5C5b20B19B301A6c28cD5060114176Cfc191D5 0x9f8a67886345fd46D3163634b57BEC47D8BB2875 0xaA1A80580B5a9586Cd6dfc24D8e94c1E57308d4c 0x3b6fE2c6AcD5B52a703a9653f4af44B1176978f4 
var poolUrl = "stratum+tcp://rbajollari.contract5:@stratum.slushpool.com:3333"

func TestHashrateContractCreation(t *testing.T) {
	// hashrate contract params
	price := 0
	limit := 0
	speed := 10000000000000
	length := 1800

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
