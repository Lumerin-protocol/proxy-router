package contractmanager

import (
	"testing"
	"time"

	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/data"
	"gitlab.com/TitanInd/hashrouter/interop"
	"gitlab.com/TitanInd/hashrouter/lib"
)

func contractManagerSetup() (*data.Collection[IContractModel], *ContractManager) {
	Setup()
	contractCollection := NewContractCollection()
	return contractCollection,
		NewContractManager(blockchainGateway, globalScheduler, log, contractCollection, interop.AddressStringToSlice(""), "", false, 0, 1*time.Second, lib.Dest{}, 30*time.Second)
}

func TestContractShouldExist(t *testing.T) {
	collection, contractManager := contractManagerSetup()

	collection.Store(&BTCHashrateContractSeller{data: blockchain.ContractData{Addr: interop.AddressStringToSlice("0x1")}})

	contractExists := contractManager.ContractExists(interop.AddressStringToSlice("0x1"))

	if !contractExists {
		t.Errorf("Expected contract to exist")
	}
}

func TestContractShouldNotExist(t *testing.T) {
	collection, contractManager := contractManagerSetup()

	collection.Store(&BTCHashrateContractSeller{data: blockchain.ContractData{Addr: interop.AddressStringToSlice("0x1")}})
	contractExists := contractManager.ContractExists(interop.AddressStringToSlice("0x2"))

	if contractExists {
		t.Errorf("Expected contract to not exist")
	}
}
