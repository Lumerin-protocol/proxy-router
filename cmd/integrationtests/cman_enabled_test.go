package integrationtests

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"gitlab.com/TitanInd/lumerin/cmd/connectionscheduler"
	"gitlab.com/TitanInd/lumerin/cmd/contractmanager"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/protocol/stratumv1"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

type EnabledConfig struct {
	BuyerNode           bool
	ListenIP            string
	ListenPort          string
	DefaultPoolAddr     string
	SchedulePassthrough bool
	Mnemonic            string
	AccountIndex        int
	EthNodeAddr         string
	ClaimFunds          bool
	TimeThreshold       int
	CloneFactoryAddress string
	LumerinTokenAddress string
	ValidatorAddress    string
	ProxyAddress        string
	LogFilePath         string
}

func LoadEnabledTestConfiguration(filePath string) (configs EnabledConfig, err error) {
	var data map[string]interface{}
	currDir, _ := os.Getwd()
	defer os.Chdir(currDir)

	if err != nil {
		panic(fmt.Errorf("error retrieving config file variable: %s", err))
	}
	file := filepath.Base(filePath)
	filePath = filepath.Dir(filePath)
	os.Chdir(filePath)

	configFile, err := os.Open(file)
	if err != nil {
		return configs, err
	}
	defer configFile.Close()
	byteValue, _ := ioutil.ReadAll(configFile)

	err = json.Unmarshal(byteValue, &data)

	configData := data["config"].(map[string]interface{})
	configs.BuyerNode = configData["buyerNode"].(bool)

	connConfigData := data["connection"].(map[string]interface{})
	configs.ListenIP = connConfigData["listenIP"].(string)
	configs.ListenPort = connConfigData["listenPort"].(string)
	configs.DefaultPoolAddr = connConfigData["defaultPoolAddr"].(string)

	contConfigData := data["contract"].(map[string]interface{})
	configs.Mnemonic = contConfigData["mnemonic"].(string)
	configs.AccountIndex = int(contConfigData["accountIndex"].(float64))
	configs.EthNodeAddr = contConfigData["ethNodeAddr"].(string)
	configs.ClaimFunds = contConfigData["claimFunds"].(bool)
	configs.TimeThreshold = int(contConfigData["timeThreshold"].(float64))
	configs.ValidatorAddress = contConfigData["validatorAddress"].(string)

	schedConfigData := data["schedule"].(map[string]interface{})
	configs.SchedulePassthrough = schedConfigData["passthrough"].(bool)

	logConfigData := data["logging"].(map[string]interface{})
	configs.LogFilePath = logConfigData["filePath"].(string)

	return configs, err
}

func EnabledSimMain(ps *msgbus.PubSub, configs EnabledConfig) (msgbus.DestID, contractmanager.SellerContractManager) {
	mainContext := context.Background()

	//
	// Add the various Context variables here
	// msgbus, logger, defailt listen address, defalt desitnation address
	//

	src := lumerinlib.NewNetAddr(lumerinlib.TCP, configs.ListenIP+":"+configs.ListenPort)
	dst := lumerinlib.NewNetAddr(lumerinlib.TCP, configs.DefaultPoolAddr)

	//
	// the proro argument (#1) gets set in the Protocol sus-system
	//
	cs := contextlib.NewContextStruct(nil, ps, nil, src, dst)

	//
	//  All of the various needed subsystem values get passed into the context here.
	//
	mainContext = context.WithValue(mainContext, contextlib.ContextKey, cs)

	//
	// Setup Default Dest
	//
	dest := &msgbus.Dest{
		ID:     msgbus.DestID(msgbus.DEFAULT_DEST_ID),
		NetUrl: msgbus.DestNetUrl(configs.DefaultPoolAddr),
	}

	event, err := ps.PubWait(msgbus.DestMsg, msgbus.IDString(msgbus.DEFAULT_DEST_ID), dest)
	if err != nil {
		panic(fmt.Sprintf("Adding Default Dest Failed: %s", err))
	}
	if event.Err != nil {
		panic(fmt.Sprintf("Adding Default Dest Failed: %s", event.Err))
	}

	//
	// Setup Node Operator Msg
	//
	nodeOperator := msgbus.NodeOperator{
		ID:          msgbus.NodeOperatorID(msgbus.GetRandomIDString()),
		IsBuyer:     configs.BuyerNode,
		DefaultDest: dest.ID,
	}
	event, err = ps.PubWait(msgbus.NodeOperatorMsg, msgbus.IDString(nodeOperator.ID), nodeOperator)
	if err != nil {
		panic(fmt.Sprintf("Adding Node Operator Failed: %s", err))
	}
	if event.Err != nil {
		panic(fmt.Sprintf("Adding Node Operator Failed: %s", event.Err))
	}

	//
	// Fire up the StratumV1 Potocol
	//
	srcStrat, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", configs.ListenIP, configs.ListenPort))
	if err != nil {
		lumerinlib.PanicHere("")
	}

	stratum, err := stratumv1.NewListener(mainContext, srcStrat, dest)
	if err != nil {
		panic(fmt.Sprintf("Stratum Protocol New() failed:%s", err))
	}

	stratum.Run()

	//
	// Fire up schedule manager
	//
	csched, err := connectionscheduler.New(&mainContext, &nodeOperator, configs.SchedulePassthrough)
	if err != nil {
		panic(fmt.Sprintf("Schedule manager failed: %v", err))
	}
	err = csched.Start()
	if err != nil {
		panic(fmt.Sprintf("Schedule manager failed to start: %v", err))
	}

	//
	// Fire up contract manager
	//
	var contractManagerConfig msgbus.ContractManagerConfig

	contractManagerConfig.ID = msgbus.ContractManagerConfigID(msgbus.GetRandomIDString())
	contractManagerConfig.Mnemonic = configs.Mnemonic
	contractManagerConfig.AccountIndex = configs.AccountIndex
	contractManagerConfig.EthNodeAddr = configs.EthNodeAddr
	contractManagerConfig.ClaimFunds = configs.ClaimFunds
	contractManagerConfig.CloneFactoryAddress = configs.CloneFactoryAddress
	contractManagerConfig.LumerinTokenAddress = configs.LumerinTokenAddress
	contractManagerConfig.ValidatorAddress = configs.ValidatorAddress
	contractManagerConfig.ProxyAddress = configs.ProxyAddress

	// Publish Contract Manager Config to MsgBus
	ps.PubWait(msgbus.ContractManagerConfigMsg, msgbus.IDString(contractManagerConfig.ID), contractManagerConfig)

	var sellerCM contractmanager.SellerContractManager
	err = contractmanager.Run(&mainContext, &sellerCM, msgbus.IDString(contractManagerConfig.ID), &nodeOperator)

	if err != nil {
		panic(fmt.Sprintf("Contract manager failed to run: %v", err))
	}

	return dest.ID, sellerCM
}

func TestEnabled(t *testing.T) {
	configPath := "../../ganacheconfig.json"

	var hashrateContractAddresses []msgbus.ContractID
	var purchasedHashrateContractAddresses []msgbus.ContractID

	targetDest1Url := "stratum+tcp://pool-east.staging.pool.titan.io:4242"
	targetDest2Url := "stratum+tcp://pool-east.staging.pool.titan.io:4242"

	configs, err := LoadEnabledTestConfiguration(configPath)
	if err != nil {
		panic(fmt.Sprintf("Loading Config Failed: %s", err))
	}

	ts, ltransaction, cftransaction := contractmanager.BeforeEach(configPath)
	configs.LumerinTokenAddress = ts.LumerinAddress.String()
	configs.CloneFactoryAddress = ts.CloneFactoryAddress.String()

	contractLength := 100
	if configs.EthNodeAddr == "ws://127.0.0.1:7545" {
		contractLength = 20 // when running in ganache
	}

	var sleepTime time.Duration = 10 * time.Second

	ps := msgbus.New(10, nil)

	// wait until transaction for deploying contracts went through before continuing
	_, lerr := ts.EthClient.TransactionReceipt(context.Background(), ltransaction.Hash())
	_, cferr := ts.EthClient.TransactionReceipt(context.Background(), cftransaction.Hash())
	for lerr != nil && cferr != nil {
		_, lerr = ts.EthClient.TransactionReceipt(context.Background(), ltransaction.Hash())
		_, cferr = ts.EthClient.TransactionReceipt(context.Background(), cftransaction.Hash())
		time.Sleep(time.Second * 10)
	}

	defaultDestID, cm := EnabledSimMain(ps, configs)

	Account, PrivateKey := contractmanager.HdWalletKeys(configs.Mnemonic, configs.AccountIndex+1)
	buyerAddress := Account.Address
	buyerPrivateKey := PrivateKey

	// subcribe to creation events emitted by clonefactory contract
	cfLogs, cfSub, _ := contractmanager.SubscribeToContractEvents(cm.EthClient, cm.CloneFactoryAddress)
	// create event signature to parse out creation, purchase, and close event
	contractCreatedSig := []byte("contractCreated(address,string)")
	contractCreatedSigHash := crypto.Keccak256Hash(contractCreatedSig)
	clonefactoryContractPurchasedSig := []byte("clonefactoryContractPurchased(address)")
	clonefactoryContractPurchasedSigHash := crypto.Keccak256Hash(clonefactoryContractPurchasedSig)

	i := 0
	j := 0
	go func() {
		for {
			select {
			case err := <-cfSub.Err():
				panic(fmt.Sprintf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err))
			case cfLog := <-cfLogs:
				switch {
				case cfLog.Topics[0].Hex() == contractCreatedSigHash.Hex():
					hashrateContractAddresses = append(hashrateContractAddresses, msgbus.ContractID(common.HexToAddress(cfLog.Topics[1].Hex()).String())) 
					fmt.Printf("Address of created Hashrate Contract: %s\n\n", hashrateContractAddresses[i])
					i++

				case cfLog.Topics[0].Hex() == clonefactoryContractPurchasedSigHash.Hex():
					purchasedHashrateContractAddresses = append(purchasedHashrateContractAddresses, msgbus.ContractID(common.HexToAddress(cfLog.Topics[1].Hex()).String()))
					fmt.Printf("Address of purchased Hashrate Contract: %s\n\n", purchasedHashrateContractAddresses[j])
					j++
				}
			}
		}
	}()

	//
	// miner connecting to lumerin node
	//
	fmt.Print("\n\n/// Miner connecting to node ///\n\n\n")

	miner := msgbus.Miner{
		ID:                   msgbus.MinerID("MinerID01"),
		IP:                   "IpAddress1",
		State:                msgbus.OnlineState,
		Dest:                 defaultDestID,
		CurrentHashRate: 	  20,
		CsMinerHandlerIgnore: false,
	}

	time.Sleep(sleepTime)

	_ = miner
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner.ID), miner)

	//
	// seller created contract found by lumerin node
	//
	fmt.Print("\n\n/// Created contracts found by lumerin node ///\n\n\n")

	contractmanager.CreateHashrateContract(cm.EthClient, cm.Account, cm.PrivateKey, cm.CloneFactoryAddress, 0, 10, 20, contractLength, common.HexToAddress(configs.ValidatorAddress))
	contractmanager.CreateHashrateContract(cm.EthClient, cm.Account, cm.PrivateKey, cm.CloneFactoryAddress, 0, 10, 60, contractLength, common.HexToAddress(configs.ValidatorAddress))

	time.Sleep(sleepTime)

	miners, _ := ps.MinerGetAllWait()
	for _, v := range miners {
		miner, _ := ps.MinerGetWait(msgbus.MinerID(v))
		if miner.Contract != "" || miner.Dest != defaultDestID {
			t.Errorf("Miner contract and dest not set correctly")
		}
	}

	// wait until created hashrate contracts were found before continuing
loop1:
	for {
		if hashrateContractAddresses[0] != "" && hashrateContractAddresses[1] != "" {
			break loop1
		}
	}
	time.Sleep(time.Second * 2)

	//
	// contract 1 was purchased and target dest was inputed in it
	//
	fmt.Print("\n\n/// Contract 1 was purchased and target dest was inputed in it ///\n\n\n")

	contractmanager.PurchaseHashrateContract(cm.EthClient, buyerAddress, buyerPrivateKey, cm.CloneFactoryAddress, common.HexToAddress(string(hashrateContractAddresses[0])), buyerAddress, targetDest1Url)
	time.Sleep(sleepTime)
	
	// wait until hashrate contract was purchased before continuing
loop2:
	for {
		if purchasedHashrateContractAddresses[0] != "" {
			break loop2
		}
	}
	time.Sleep(time.Second * 2)

	// find dest published by contractmanager
	var targetDest1 msgbus.DestID
	event, err := ps.GetWait(msgbus.DestMsg, "")
	if err != nil {
		panic(fmt.Sprintf("Getting all dests failed: %s", err))
	}
	for _, v := range event.Data.(msgbus.IDIndex) {
		dest, err := ps.DestGetWait(msgbus.DestID(v))
		if err != nil {
			panic(fmt.Sprintf("Getting dest failed: %s", err))
		}
		if dest.NetUrl == msgbus.DestNetUrl(targetDest1Url) {
			targetDest1 = msgbus.DestID(v)
		}
	}

	if targetDest1 == "" {
		t.Errorf("Contract manager did not publish target dest after contract was purchased")
	}

	miners, _ = ps.MinerGetAllWait()
	for _, v := range miners {
		miner, _ := ps.MinerGetWait(msgbus.MinerID(v))
		if miner.Contract != hashrateContractAddresses[0] || miner.Dest != targetDest1 {
			t.Errorf("Miner contract and dest not set correctly")
		}
	}

	//
	// More miners connecting to node
	//
	fmt.Print("\n\n/// More miners connection to node ///\n\n\n")
	miner2 := msgbus.Miner{
		ID:                   msgbus.MinerID("MinerID02"),
		IP:                   "IpAddress2",
		State:                msgbus.OnlineState,
		Dest:                 defaultDestID,
		CurrentHashRate: 	  10,
		CsMinerHandlerIgnore: false,
	}
	miner3 := msgbus.Miner{
		ID:                   msgbus.MinerID("MinerID03"),
		IP:                   "IpAddress3",
		State:                msgbus.OnlineState,
		Dest:                 defaultDestID,
		CurrentHashRate: 	  50,
		CsMinerHandlerIgnore: false,
	}

	_ = miner2
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner2.ID), miner2)
	_ = miner3
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner3.ID), miner3)
	time.Sleep(sleepTime)

	//
	// contract 2 was purchased and target dest was inputed in it
	//
	fmt.Print("\n\n/// Contract 2 was purchased and target dest 2 was inputed in it ///\n\n\n")

	contractmanager.PurchaseHashrateContract(cm.EthClient, buyerAddress, buyerPrivateKey, cm.CloneFactoryAddress, common.HexToAddress(string(hashrateContractAddresses[1])), buyerAddress, targetDest2Url)
	time.Sleep(sleepTime)
	
	// wait until hashrate contract was purchased before continuing
loop3:
	for {
		if purchasedHashrateContractAddresses[1] != "" {
			break loop3
		}
	}
	time.Sleep(time.Second * 2)

	// find dest published by contractmanager
	var targetDest2 msgbus.DestID
	event, err = ps.GetWait(msgbus.DestMsg, "")
	if err != nil {
		panic(fmt.Sprintf("Getting all dests failed: %s", err))
	}
	for _, v := range event.Data.(msgbus.IDIndex) {
		dest, err := ps.DestGetWait(msgbus.DestID(v))
		if err != nil {
			panic(fmt.Sprintf("Getting dest failed: %s", err))
		}
		if dest.NetUrl == msgbus.DestNetUrl(targetDest2Url) {
			targetDest2 = msgbus.DestID(v)
		}
	}

	if targetDest2 == "" {
		t.Errorf("Contract manager did not publish target dest after contract was purchased")
	}

	//
	// contract length expires
	//
	fmt.Print("\n\n/// Contract length expires ///\n\n\n")

	// if network is ganache, create a new transaction so a new block is created
	if configs.EthNodeAddr == "ws://127.0.0.1:7545" {
		contractmanager.CreateNewGanacheBlock(ts, cm.Account, cm.PrivateKey, contractLength, 0)
	}

	// subcribe to contract closed event emitted by hashrate contract
	hrLogs, hrSub, _ := contractmanager.SubscribeToContractEvents(cm.EthClient, common.HexToAddress(string(hashrateContractAddresses[1])))
	// create event signature to parse out creation, purchase, and close event
	contractClosedSig := []byte("contractClosed()")
	contractClosedSigHash := crypto.Keccak256Hash(contractClosedSig)
loop4:
	for {
		select {
		case err := <-hrSub.Err():
			panic(fmt.Sprintf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err))
		case hrLog := <-hrLogs:
			if hrLog.Topics[0].Hex() == contractClosedSigHash.Hex() {
				break loop4
			}
		}
	}

	time.Sleep(2 * time.Second)

	// contract back to available
	event, err = ps.GetWait(msgbus.ContractMsg, msgbus.IDString(hashrateContractAddresses[0]))
	if err != nil {
		panic(fmt.Sprintf("Getting contract failed: %s", err))
	}
	contract := event.Data.(msgbus.Contract)
	if contract.State != msgbus.ContAvailableState || cm.NodeOperator.Contracts[hashrateContractAddresses[0]] != msgbus.ContAvailableState {
		t.Errorf("Contract not back to available")
	}

	miners, _ = ps.MinerGetAllWait()
	for _, v := range miners {
		miner, _ := ps.MinerGetWait(msgbus.MinerID(v))
		if miner.Contract != "" || miner.Dest != defaultDestID {
			t.Errorf("Miner contract and dest not set correctly")
		}
	}
}