package externalapi

import (
	"fmt"
	"testing"
	"time"

	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus/msgdata"
)

func TestMsgBusDataAddedToApiRepos(t *testing.T) {
	l := log.New()
	ps := msgbus.New(10, l)

	dest := msgbus.Dest{
		ID:     msgbus.DestID("DestID01"),
		NetUrl: msgbus.DestNetUrl("stratum+tcp://127.0.0.1:3334/"),
	}
	nodeOperator := msgbus.NodeOperator{
		ID:                     msgbus.NodeOperatorID("NodeOperatorID01"),
		DefaultDest:            dest.ID,
		TotalAvailableHashRate: 0,
		UnusedHashRate:         0,
	}
	contract := msgbus.Contract{
		IsSeller:               true,
		ID:                     msgbus.ContractID("ContractID01"),
		State:                  msgbus.ContRunningState,
		Buyer:                  "BuyerID01",
		Price:                  100,
		Limit:                  100,
		Speed:                  100,
		Length:                 100,
		StartingBlockTimestamp: 100,
		Dest:                   dest.ID,
	}
	nodeOperator.Contracts = map[msgbus.ContractID]msgbus.ContractState{
		contract.ID: msgbus.ContRunningState,
	}
	config := msgbus.ConfigInfo{
		ID:           msgbus.ConfigID("ConfigID01"),
		DefaultDest:  dest.ID,
		NodeOperator: nodeOperator.ID,
	}
	miner := msgbus.Miner{
		ID:                      msgbus.MinerID("MinerID01"),
		State:                   msgbus.OnlineState,
		Dest:                    dest.ID,
		InitialMeasuredHashRate: 10000,
		CurrentHashRate:         9000,
	}
	connection := msgbus.Connection{
		ID:        msgbus.ConnectionID("ConnectionID01"),
		Miner:     miner.ID,
		Dest:      dest.ID,
		State:     msgbus.ConnAuthState,
		TotalHash: 10000,
		StartDate: time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC),
	}

	api := New(ps)
	time.Sleep(time.Millisecond * 1000)

	go api.Run("8080", l)

	fmt.Print("\n/// Publish Msgbus Msgs //\n\n")
	ps.PubWait(msgbus.DestMsg, msgbus.IDString(dest.ID), msgbus.Dest{})
	time.Sleep(time.Millisecond * 100)
	if api.Dest.DestJSONs[0].ID != string(dest.ID) {
		t.Errorf("Failed to add dest to API repo after publish from msgbus")
	}
	ps.PubWait(msgbus.NodeOperatorMsg, msgbus.IDString(nodeOperator.ID), msgbus.NodeOperator{})
	time.Sleep(time.Millisecond * 100)
	if api.NodeOperator.NodeOperatorJSONs[0].ID != string(nodeOperator.ID) {
		t.Errorf("Failed to add nodeOperator to API repo after publish from msgbus")
	}
	ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contract.ID), msgbus.Contract{})
	time.Sleep(time.Millisecond * 100)
	if api.Contract.ContractJSONs[0].ID != string(contract.ID) {
		t.Errorf("Failed to add contract to API repo after publish from msgbus")
	}
	ps.PubWait(msgbus.ConfigMsg, msgbus.IDString(config.ID), msgbus.ConfigInfo{})
	time.Sleep(time.Millisecond * 100)
	if api.Config.ConfigInfoJSONs[0].ID != string(config.ID) {
		t.Errorf("Failed to add config to API repo after publish from msgbus")
	}
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner.ID), msgbus.Miner{})
	time.Sleep(time.Millisecond * 100)
	if api.Miner.MinerJSONs[0].ID != string(miner.ID) {
		t.Errorf("Failed to add miner to API repo after publish from msgbus")
	}
	ps.PubWait(msgbus.ConnectionMsg, msgbus.IDString(connection.ID), msgbus.Connection{})
	time.Sleep(time.Millisecond * 100)
	if api.Connection.ConnectionJSONs[0].ID != string(connection.ID) {
		t.Errorf("Failed to add connection to API repo after publish from msgbus")
	}
	fmt.Print("\nAPI Repos::\n")
	fmt.Println("Dest Repo: ", api.Dest.DestJSONs)
	fmt.Println("NodeOperator Repo: ", api.NodeOperator.NodeOperatorJSONs)
	fmt.Println("Contract Repo: ", api.Contract.ContractJSONs)
	fmt.Println("Config Repo: ", api.Config.ConfigInfoJSONs)
	fmt.Println("Miner Repo: ", api.Miner.MinerJSONs)
	fmt.Println("Connection Repo: ", api.Connection.ConnectionJSONs)

	fmt.Print("\n/// Update Msgbus Msgs //\n\n")
	ps.SetWait(msgbus.DestMsg, msgbus.IDString(dest.ID), dest)
	time.Sleep(time.Millisecond * 100)
	if api.Dest.DestJSONs[0] != msgdata.ConvertDestMSGtoDestJSON(dest) {
		t.Errorf("Failed to update dest in API repo after update from msgbus")
	}
	ps.SetWait(msgbus.NodeOperatorMsg, msgbus.IDString(nodeOperator.ID), nodeOperator)
	time.Sleep(time.Millisecond * 100)
	if api.NodeOperator.NodeOperatorJSONs[0].DefaultDest != msgdata.ConvertNodeOperatorMSGtoNodeOperatorJSON(nodeOperator).DefaultDest {
		t.Errorf("Failed to update nodeOperator in API repo after update from msgbus")
	}
	ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contract.ID), contract)
	time.Sleep(time.Millisecond * 100)
	if api.Contract.ContractJSONs[0] != msgdata.ConvertContractMSGtoContractJSON(contract) {
		t.Errorf("Failed to update contract in API repo after update from msgbus")
	}
	ps.SetWait(msgbus.ConfigMsg, msgbus.IDString(config.ID), config)
	time.Sleep(time.Millisecond * 100)
	if api.Config.ConfigInfoJSONs[0] != msgdata.ConvertConfigInfoMSGtoConfigInfoJSON(config) {
		t.Errorf("Failed to update config in API repo after update from msgbus")
	}
	ps.SetWait(msgbus.MinerMsg, msgbus.IDString(miner.ID), miner)
	time.Sleep(time.Millisecond * 100)
	if api.Miner.MinerJSONs[0] != msgdata.ConvertMinerMSGtoMinerJSON(miner) {
		t.Errorf("Failed to update miner in API repo after update from msgbus")
	}
	ps.SetWait(msgbus.ConnectionMsg, msgbus.IDString(connection.ID), connection)
	time.Sleep(time.Millisecond * 100)
	if api.Connection.ConnectionJSONs[0] != msgdata.ConvertConnectionMSGtoConnectionJSON(connection) {
		t.Errorf("Failed to update connection in API repo after update from msgbus")
	}
	fmt.Print("\nAPI Repos::\n")
	fmt.Println("Dest Repo: ", api.Dest.DestJSONs)
	fmt.Println("NodeOperator Repo: ", api.NodeOperator.NodeOperatorJSONs)
	fmt.Println("Contract Repo: ", api.Contract.ContractJSONs)
	fmt.Println("Config Repo: ", api.Config.ConfigInfoJSONs)
	fmt.Println("Miner Repo: ", api.Miner.MinerJSONs)
	fmt.Println("Connection Repo: ", api.Connection.ConnectionJSONs)

	time.Sleep(time.Minute / 6)
	fmt.Print("\n/// UnPublish Msgbus Msgs //\n\n")
	ps.UnpubWait(msgbus.DestMsg, msgbus.IDString(dest.ID))
	time.Sleep(time.Millisecond * 100)
	if len(api.Dest.DestJSONs) > 0 {
		t.Errorf("Failed to remove dest from API repo after unpublish from msgbus")
	}
	ps.UnpubWait(msgbus.NodeOperatorMsg, msgbus.IDString(nodeOperator.ID))
	time.Sleep(time.Millisecond * 100)
	if len(api.NodeOperator.NodeOperatorJSONs) > 0 {
		t.Errorf("Failed to remove nodeOperator from API repo after unpublish from msgbus")
	}
	ps.UnpubWait(msgbus.ContractMsg, msgbus.IDString(contract.ID))
	time.Sleep(time.Millisecond * 100)
	if len(api.Contract.ContractJSONs) > 0 {
		t.Errorf("Failed to remove contract from API repo after unpublish from msgbus")
	}
	ps.UnpubWait(msgbus.ConfigMsg, msgbus.IDString(config.ID))
	time.Sleep(time.Millisecond * 100)
	if len(api.Config.ConfigInfoJSONs) > 0 {
		t.Errorf("Failed to remove config from API repo after unpublish from msgbus")
	}
	ps.UnpubWait(msgbus.MinerMsg, msgbus.IDString(miner.ID))
	time.Sleep(time.Millisecond * 100)
	if len(api.Miner.MinerJSONs) > 0 {
		t.Errorf("Failed to remove miner from API repo after unpublish from msgbus")
	}
	ps.UnpubWait(msgbus.ConnectionMsg, msgbus.IDString(connection.ID))
	time.Sleep(time.Millisecond * 100)
	if len(api.Connection.ConnectionJSONs) > 0 {
		t.Errorf("Failed to remove connection from API repo after unpublish from msgbus")
	}
	fmt.Print("\nAPI Repos::\n")
	fmt.Println("Dest Repo: ", api.Dest.DestJSONs)
	fmt.Println("NodeOperator Repo: ", api.NodeOperator.NodeOperatorJSONs)
	fmt.Println("Contract Repo: ", api.Contract.ContractJSONs)
	fmt.Println("Config Repo: ", api.Config.ConfigInfoJSONs)
	fmt.Println("Miner Repo: ", api.Miner.MinerJSONs)
	fmt.Println("Connection Repo: ", api.Connection.ConnectionJSONs)
	time.Sleep(time.Minute / 6)
}
