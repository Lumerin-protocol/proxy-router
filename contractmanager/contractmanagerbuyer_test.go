package contractmanager

import (
	//"crypto/ecdsa"
	//"crypto/rand"
	//"errors"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	//"github.com/ethereum/go-ethereum/crypto/ecies"

	"gitlab.com/TitanInd/lumerin/cmd/connectionscheduler"
	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/connections"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

func TestBuyerRoutine(t *testing.T) {
	configPath := "../../ganacheconfig.json"
	mnemonic := "course surface achieve episode cable brisk flame enjoy beyond hand rival predict"
	accountIndex := 0
	l := log.New()
	ps := msgbus.New(10, l)
	ts, _, _ := BeforeEach(configPath, mnemonic)
	var hashrateContractAddress [3]common.Address
	var purchasedHashrateContractAddress [3]common.Address

	ctxStruct := contextlib.NewContextStruct(nil, ps, nil, nil, nil)
	mainCtx := context.WithValue(context.Background(), contextlib.ContextKey, ctxStruct)

	var contractManagerConfig lumerinlib.ContractManagerConfig

	contractLength := 10000

	defaultpooladdr := "stratum+tcp://127.0.0.1:33334/"
	defaultDest := msgbus.Dest{
		ID:     msgbus.DestID(msgbus.DEFAULT_DEST_ID),
		NetUrl: msgbus.DestNetUrl(defaultpooladdr),
	}
	event, err := ps.PubWait(msgbus.DestMsg, msgbus.IDString(msgbus.DEFAULT_DEST_ID), defaultDest)
	if err != nil {
		panic(fmt.Sprintf("Adding Default Dest Failed: %s", err))
	}
	if event.Err != nil {
		panic(fmt.Sprintf("Adding Default Dest Failed: %s", event.Err))
	}

	contractManagerConfigFile, err := LoadTestConfiguration("contract", configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load contract manager configuration:%s", err))
	}

	contractManagerConfig.Mnemonic = mnemonic
	contractManagerConfig.AccountIndex = accountIndex
	contractManagerConfig.TimeThreshold = int(contractManagerConfigFile["timeThreshold"].(float64))
	contractManagerConfig.EthNodeAddr = contractManagerConfigFile["ethNodeAddr"].(string)
	contractManagerConfig.CloneFactoryAddress = ts.CloneFactoryAddress.Hex()

	sleepTime := 5000 // 5000 ms sleeptime in ganache
	if contractManagerConfig.EthNodeAddr != "ws://127.0.0.1:7545" {
		sleepTime = 30000 // 20000 ms on testnet
	}

	Account, PrivateKey := HdWalletKeys(contractManagerConfig.Mnemonic, contractManagerConfig.AccountIndex+1)
	sellerAddress := Account.Address
	sellerPrivateKey := PrivateKey

	NodeOperator := msgbus.NodeOperator{
		ID:          msgbus.NodeOperatorID(msgbus.GetRandomIDString()),
		DefaultDest: defaultDest.ID,
		IsBuyer:     true,
		Contracts:   make(map[string]string),
	}
	event, err = ps.PubWait(msgbus.NodeOperatorMsg, msgbus.IDString(NodeOperator.ID), NodeOperator)
	if err != nil {
		panic(fmt.Sprintf("Adding Node Operator Failed: %s", err))
	}
	if event.Err != nil {
		panic(fmt.Sprintf("Adding Node Operator Failed: %s", event.Err))
	}

	// start connection scheduler look at miners
	connectionCollection := connections.CreateConnectionCollection()
	cs, err := connectionscheduler.New(&mainCtx, &NodeOperator, false, 0, connectionCollection)
	if err != nil {
		panic(fmt.Sprintf("schedule manager failed:%s", err))
	}
	err = cs.Start()
	if err != nil {
		panic(fmt.Sprintf("schedule manager failed to start:%s", err))
	}

	var cman BuyerContractManager
	err = cman.init(&mainCtx, contractManagerConfig, &NodeOperator)
	if err != nil {
		panic(fmt.Sprintf("contract manager init failed:%s", err))
	}

	// subcribe to creation events emitted by clonefactory contract
	cfLogs, cfSub, _ := SubscribeToContractEvents(ts.EthClient, ts.CloneFactoryAddress)
	// create event signature to parse out creation event
	contractCreatedSig := []byte("contractCreated(address,string)")
	contractCreatedSigHash := crypto.Keccak256Hash(contractCreatedSig)
	clonefactoryContractPurchasedSig := []byte("clonefactoryContractPurchased(address)")
	clonefactoryContractPurchasedSigHash := crypto.Keccak256Hash(clonefactoryContractPurchasedSig)
	go func() {
		i := 0
		j := 0
		for {
			select {
			case err := <-cfSub.Err():
				panic(fmt.Sprintf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err))
			case cfLog := <-cfLogs:
				switch {
				case cfLog.Topics[0].Hex() == contractCreatedSigHash.Hex():
					hashrateContractAddress[i] = common.HexToAddress(cfLog.Topics[1].Hex())
					fmt.Printf("Address of created Hashrate Contract %d: %s\n\n", i+1, hashrateContractAddress[i].Hex())
					i++

				case cfLog.Topics[0].Hex() == clonefactoryContractPurchasedSigHash.Hex():
					purchasedHashrateContractAddress[j] = common.HexToAddress(cfLog.Topics[1].Hex())
					fmt.Printf("Address of purchased Hashrate Contract %d: %s\n\n", j+1, purchasedHashrateContractAddress[j].Hex())
					j++
				}
			}
		}
	}()

	//
	// test startup with 1 running contract and 1 availabe contract
	//
	CreateHashrateContract(cman.EthClient, sellerAddress, sellerPrivateKey, ts.CloneFactoryAddress, int(0), int(10), int(31), int(contractLength), cman.Account)
	CreateHashrateContract(cman.EthClient, sellerAddress, sellerPrivateKey, ts.CloneFactoryAddress, int(0), int(10), int(41), int(contractLength), cman.Account)

	// wait until created hashrate contract was found before continuing
loop1:
	for {
		if hashrateContractAddress[0] != common.HexToAddress("0x0000000000000000000000000000000000000000") {
			break loop1
		}
	}
	time.Sleep(time.Millisecond * time.Duration(sleepTime/5))
	PurchaseHashrateContract(cman.EthClient, cman.Account, cman.PrivateKey, ts.CloneFactoryAddress, hashrateContractAddress[0], cman.Account, "stratum+tcp://127.0.0.1:3333/testrig")

	// wait until hashrate contract was purchased before continuing
loop2:
	for {
		if purchasedHashrateContractAddress[0] != common.HexToAddress("0x0000000000000000000000000000000000000000") {
			break loop2
		}
	}
	// publish miners sent from seller to fulfill hashrate promised by contract
	miner1 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID01"),
		IP:              "IpAddress1",
		CurrentHashRate: 20,
		State:           msgbus.OnlineState,
	}
	miner2 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID02"),
		IP:              "IpAddress2",
		CurrentHashRate: 10,
		State:           msgbus.OnlineState,
	}
	ps.Pub(msgbus.MinerMsg, msgbus.IDString(miner1.ID), miner1)
	ps.Pub(msgbus.MinerMsg, msgbus.IDString(miner2.ID), miner2)

	err = cman.start()
	if err != nil {
		panic(fmt.Sprintf("contract manager failed to start:%s", err))
	}
	if err != nil {
		panic(fmt.Sprintf("contract manager failed to start:%s", err))
	}

	// contract manager sees existing contracts and states are correct
	if cman.NodeOperator.Contracts[string(hashrateContractAddress[0].Hex())] != msgbus.ContRunningState {
		t.Errorf("Contract 1 was not found or is not in correct state")
	}
	if _, ok := cman.NodeOperator.Contracts[string(hashrateContractAddress[1].Hex())]; ok {
		t.Errorf("Contract 2 was found by buyer node while in the available state")
	}

	// connection scheduler sets contract to correct miners
	m1, _ := ps.MinerGetWait(miner1.ID)
	m2, _ := ps.MinerGetWait(miner2.ID)
	if _, ok := m1.Contracts[string(hashrateContractAddress[0].Hex())]; ok {
		t.Errorf("Miner contracts not set correctly")
	}
	if _, ok := m2.Contracts[string(hashrateContractAddress[0].Hex())]; ok {
		t.Errorf("Miner contracts not set correctly")
	}

	// contract manager should updated states
	// wait until created hashrate contract was found before continuing
loop3:
	for {
		if hashrateContractAddress[1] != common.HexToAddress("0x0000000000000000000000000000000000000000") {
			break loop3
		}
	}
	time.Sleep(time.Millisecond * time.Duration(sleepTime/5))
	PurchaseHashrateContract(cman.EthClient, cman.Account, cman.PrivateKey, ts.CloneFactoryAddress, hashrateContractAddress[1], cman.Account, "stratum+tcp://127.0.0.1:3333/testrig")

	// wait until hashrate contract was purchased before continuing
loop4:
	for {
		if purchasedHashrateContractAddress[1] != common.HexToAddress("0x0000000000000000000000000000000000000000") {
			break loop4
		}
	}
	time.Sleep(time.Millisecond * time.Duration(sleepTime/5))
	miner3 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID03"),
		IP:              "IpAddress3",
		CurrentHashRate: 40,
		State:           msgbus.OnlineState,
	}
	ps.Pub(msgbus.MinerMsg, msgbus.IDString(miner3.ID), miner3)
	time.Sleep(time.Millisecond * time.Duration(sleepTime))

	if cman.NodeOperator.Contracts[string(hashrateContractAddress[1].Hex())] != msgbus.ContRunningState {
		t.Errorf("Contract 2 is not in correct state")
	}

	// connection scheduler sets contracts to correct miners
	m1, _ = ps.MinerGetWait(miner1.ID)
	m2, _ = ps.MinerGetWait(miner2.ID)
	m3, _ := ps.MinerGetWait(miner3.ID)
	time.Sleep(time.Millisecond * time.Duration(sleepTime/5))
	if _, ok := m1.Contracts[string(hashrateContractAddress[0].Hex())]; ok {
		t.Errorf("Miner contracts not set correctly")
	}
	if _, ok := m2.Contracts[string(hashrateContractAddress[0].Hex())]; ok {
		t.Errorf("Miner contracts not set correctly")
	}
	if _, ok := m3.Contracts[string(hashrateContractAddress[1].Hex())]; ok {
		t.Errorf("Miner contracts not set correctly")
	}

	/*
		//
		// Test early closeout from seller
		//
		CreateHashrateContract(cman.EthClient, sellerAddress, sellerPrivateKey, ts.cloneFactoryAddress, int(0), int(0), int(30), int(contractLength*10), cman.Account)
		time.Sleep(time.Millisecond * time.Duration(sleepTime))
		if _,ok := cman.msg.Contracts[string(hashrateContractAddress[2].Hex())] ; ok {
			t.Errorf("Contract 3 was found by buyer node while in the available state")
		}

		PurchaseHashrateContract(cman.EthClient, cman.Account, cman.PrivateKey, ts.cloneFactoryAddress, hashrateContractAddress[2], cman.Account, "stratum+tcp://127.0.0.1:3333/testrig")
		time.Sleep(time.Millisecond * time.Duration(sleepTime))
		if cman.msg.Contracts[string(hashrateContractAddress[2].Hex())] != msgbus.ContRunningState {
			t.Errorf("Contract 3 is not in correct state")
		}

		var wg sync.WaitGroup
		wg.Add(1)
		setContractCloseOut(cman.EthClient, sellerAddress, sellerPrivateKey, hashrateContractAddress[2], &wg, &cman.currentNonce, 0)
		wg.Wait()
		time.Sleep(time.Millisecond * time.Duration(sleepTime))
		if _,ok := cman.msg.Contracts[string(hashrateContractAddress[2].Hex())]; ok {
			t.Errorf("Contract 3 did not close out correctly")
		}
	*/

	//
	// Test contract creation, purchasing, and target dest being updated while node is running
	//
	CreateHashrateContract(cman.EthClient, sellerAddress, sellerPrivateKey, ts.CloneFactoryAddress, int(0), int(10), int(100), int(contractLength), cman.Account)

loop5:
	for {
		if hashrateContractAddress[2] != common.HexToAddress("0x0000000000000000000000000000000000000000") {
			break loop5
		}
	}
	time.Sleep(time.Millisecond * time.Duration(sleepTime/5))
	if _, ok := cman.NodeOperator.Contracts[string(hashrateContractAddress[2].Hex())]; ok {
		t.Errorf("Contract 3 was found by buyer node while in the available state")
	}
	PurchaseHashrateContract(cman.EthClient, cman.Account, cman.PrivateKey, ts.CloneFactoryAddress, hashrateContractAddress[2], cman.Account, "stratum+tcp://127.0.0.1:3333/testrig")

	// wait until hashrate contract was purchased before continuing
loop6:
	for {
		if purchasedHashrateContractAddress[2] != common.HexToAddress("0x0000000000000000000000000000000000000000") {
			break loop6
		}
	}
	time.Sleep(time.Millisecond * time.Duration(sleepTime/5))
	miner4 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID04"),
		IP:              "IpAddress4",
		CurrentHashRate: 100,
		State:           msgbus.OnlineState,
	}
	ps.Pub(msgbus.MinerMsg, msgbus.IDString(miner4.ID), miner4)
	time.Sleep(time.Millisecond * time.Duration(sleepTime/5))

	if cman.NodeOperator.Contracts[string(hashrateContractAddress[2].Hex())] != msgbus.ContRunningState {
		t.Errorf("Contract 3 is not in correct state")
	}

	// connection scheduler sets contracts to correct miners
	m1, _ = ps.MinerGetWait(miner1.ID)
	m2, _ = ps.MinerGetWait(miner2.ID)
	m3, _ = ps.MinerGetWait(miner3.ID)
	m4, _ := ps.MinerGetWait(miner4.ID)
	time.Sleep(time.Millisecond * time.Duration(sleepTime/5))
	if _, ok := m1.Contracts[string(hashrateContractAddress[0].Hex())]; ok {
		t.Errorf("Miner contracts not set correctly")
	}
	if _, ok := m2.Contracts[string(hashrateContractAddress[0].Hex())]; ok {
		t.Errorf("Miner contracts not set correctly")
	}
	if _, ok := m3.Contracts[string(hashrateContractAddress[1].Hex())]; ok {
		t.Errorf("Miner contracts not set correctly")
	}
	if _, ok := m4.Contracts[string(hashrateContractAddress[2].Hex())]; ok {
		t.Errorf("Miner contracts not set correctly")
	}

	UpdateCipherText(cman.EthClient, cman.Account, cman.PrivateKey, hashrateContractAddress[2], "stratum+tcp://127.0.0.1:3333/updated")
	time.Sleep(time.Millisecond * time.Duration(sleepTime*2))
	// check dest msg with associated contract was updated in msgbus
	event, err = ps.GetWait(msgbus.ContractMsg, msgbus.IDString(hashrateContractAddress[2].Hex()))
	if err != nil {
		panic(fmt.Sprintf("Getting Purchased Contract Failed: %s", err))
	}
	if event.Err != nil {
		panic(fmt.Sprintf("Getting Purchased Contract Failed: %s", event.Err))
	}
	contractMsg := event.Data.(msgbus.Contract)
	event, err = ps.GetWait(msgbus.DestMsg, msgbus.IDString(contractMsg.Dest))
	if err != nil {
		panic(fmt.Sprintf("Getting Dest Failed: %s", err))
	}
	if event.Err != nil {
		panic(fmt.Sprintf("Getting Dest Failed: %s", event.Err))
	}
	destMsg := event.Data.(msgbus.Dest)
	if destMsg.NetUrl != "stratum+tcp://127.0.0.1:3333/updated" {
		t.Errorf("Contract 3's target dest was not updated")
	}

	//
	// Test miners being updated below min, deleted, and set to offline
	//
	// miner 4's hashrate is updated to below min
	miner4.CurrentHashRate = 5
	ps.Set(msgbus.MinerMsg, msgbus.IDString(miner1.ID), miner1)
	time.Sleep(time.Millisecond * time.Duration(sleepTime*3))

	// miner 2 deleted
	ps.UnpubWait(msgbus.MinerMsg, msgbus.IDString(miner2.ID))
	time.Sleep(time.Millisecond * time.Duration(sleepTime*3))

	//
	// Test miners are set to offline state so running contracts should close out
	//
	miner1.State = msgbus.OfflineState
	ps.Set(msgbus.MinerMsg, msgbus.IDString(miner1.ID), miner1)
	miner3.State = msgbus.OfflineState
	ps.Set(msgbus.MinerMsg, msgbus.IDString(miner3.ID), miner3)
	miner4.State = msgbus.OfflineState
	ps.Set(msgbus.MinerMsg, msgbus.IDString(miner4.ID), miner4)
	time.Sleep(time.Millisecond * time.Duration(sleepTime*4))

	// check contracts map is empty now
	if len(cman.NodeOperator.Contracts) != 0 {
		t.Errorf("Contracts did not closeout after all miners were set to offline")
	}

	// connection scheduler removes contracts from miners
	m1, _ = ps.MinerGetWait(miner1.ID)
	m3, _ = ps.MinerGetWait(miner3.ID)
	m4, _ = ps.MinerGetWait(miner4.ID)
	if len(m1.Contracts) == 0 {
		t.Errorf("Miner contracts not set correctly")
	}
	if len(m3.Contracts) == 0 {
		t.Errorf("Miner contracts not set correctly")
	}
	if len(m4.Contracts) == 0 {
		t.Errorf("Miner contracts not set correctly")
	}
}
