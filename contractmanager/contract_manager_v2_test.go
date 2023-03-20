package contractmanager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/TitanInd/hashrouter/contractmanager/contractdata"
	"gitlab.com/TitanInd/hashrouter/data"
	"gitlab.com/TitanInd/hashrouter/interop"
	"gitlab.com/TitanInd/hashrouter/lib"
)

func contractManagerSetup() (*data.Collection[IContractModel], *ContractManager) {
	Setup()
	contractCollection := NewContractCollection()
	globalHashrate := NewGlobalHashrate()
	return contractCollection,
		NewContractManager(blockchainGateway, globalScheduler, globalHashrate, log, contractCollection, interop.AddressStringToSlice(""), "", false, 0, 1*time.Second, lib.Dest{}, 30*time.Second, 7*time.Minute)
}

func TestContractShouldExist(t *testing.T) {
	collection, contractManager := contractManagerSetup()

	data := contractdata.GetSampleContractDataDecrypted()
	collection.Store(&BTCHashrateContractSeller{data: data})

	contractExists := contractManager.ContractExists(data.Addr)

	assert.True(t, contractExists, "expected contract to exist")
}

func TestContractShouldNotExist(t *testing.T) {
	collection, contractManager := contractManagerSetup()

	data := contractdata.GetSampleContractDataDecrypted()
	collection.Store(&BTCHashrateContractSeller{data: data})
	contractExists := contractManager.ContractExists(lib.GetRandomAddr())

	if contractExists {
		t.Errorf("Expected contract to not exist")
	}
}
