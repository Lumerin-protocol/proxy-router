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

	"gitlab.com/TitanInd/lumerin/cmd/connectionscheduler"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/connections"
	"gitlab.com/TitanInd/lumerin/cmd/protocol/stratumv1"
	"gitlab.com/TitanInd/lumerin/cmd/validator/validator"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

type ValDisabledConfig struct {
	BuyerNode           bool
	ListenIP            string
	ListenPort          string
	DefaultPoolAddr     string
	SchedulePassthrough bool
	LogFilePath         string
}

func LoadValDisabledTestConfiguration(filePath string) (configs ValDisabledConfig, err error) {
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

	schedConfigData := data["schedule"].(map[string]interface{})
	configs.SchedulePassthrough = schedConfigData["passthrough"].(bool)

	logConfigData := data["logging"].(map[string]interface{})
	configs.LogFilePath = logConfigData["filePath"].(string)

	return configs, err
}

func ValDisabledSimMain(ps *msgbus.PubSub, configs ValDisabledConfig, hashrateCalcLagTime time.Duration) msgbus.DestID {
	mainContext := context.Background()

	//
	// Create Connection Collection
	//
	connectionCollection := connections.CreateConnectionCollection()

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
	csched, err := connectionscheduler.New(&mainContext, &nodeOperator, false, int(hashrateCalcLagTime), connectionCollection)
	if err != nil {
		panic(fmt.Sprintf("Schedule manager failed: %v", err))
	}
	err = csched.Start()
	if err != nil {
		panic(fmt.Sprintf("Schedule manager failed to start: %v", err))
	}

	//
	// Fire up validator
	//
	v := validator.MakeNewValidator(&mainContext)
	err = v.Start()
	if err != nil {
		panic(fmt.Sprintf("Validator failed to start: %v", err))
	}
	
	return dest.ID
}

func TestValDisabled(t *testing.T) {
	configPath := "../../lumerinconfig.json"

	configs, err := LoadValDisabledTestConfiguration(configPath)
	if err != nil {
		panic(fmt.Sprintf("Loading Config Failed: %s", err))
	}

	l := log.New()
	ps := msgbus.New(10, l)

	var hashrateCalcLagTime time.Duration = 20
	var reAdjustmentTime time.Duration = 3

	defaultDestID := ValDisabledSimMain(ps, configs, hashrateCalcLagTime)

	fmt.Print("\n\n/// Multiple miners connecting to node ///\n\n\n")

	miner1 := msgbus.Miner{
		ID:                   msgbus.MinerID("MinerID01"),
		IP:                   "IpAddress1",
		CurrentHashRate:      0,
		State:                msgbus.OnlineState,
		Dest:                 defaultDestID,
		Contracts: 			  make(map[msgbus.ContractID]float64),	
	}
	miner2 := msgbus.Miner{
		ID:                   msgbus.MinerID("MinerID02"),
		IP:                   "IpAddress2",
		CurrentHashRate:      0,
		State:                msgbus.OnlineState,
		Dest:                 defaultDestID,
		Contracts: 			  make(map[msgbus.ContractID]float64),
	}
	miner3 := msgbus.Miner{
		ID:                   msgbus.MinerID("MinerID03"),
		IP:                   "IpAddress3",
		CurrentHashRate:      0,
		State:                msgbus.OnlineState,
		Dest:                 defaultDestID,
		Contracts: 			  make(map[msgbus.ContractID]float64),
	}
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner1.ID), miner1)
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner2.ID), miner2)
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner3.ID), miner3)
	ps.SendValidateSetDiff(context.Background(), miner1.ID, defaultDestID, 65535)
	time.Sleep(time.Second * 2)
	ps.SendValidateSetDiff(context.Background(), miner2.ID, defaultDestID, 65535)
	time.Sleep(time.Second * 2)
	ps.SendValidateSetDiff(context.Background(), miner3.ID, defaultDestID, 65535)
	time.Sleep(time.Second * 2)

	fmt.Print("\n\n/// Validator getting submits and updating miner hashrates ///\n\n\n")

	/*
	{"id":"4","jsonrpc":"2.0","method":"mining.notify","params":["6c84558d","a1ac7997ce3af8f42a60e6982e378952456914d00004dc270000000000000000","01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff2103133d0b0004a57c81620c","5f546974616e2e696f5fffffffff0247953d26000000001976a914288913831abe556f331ed13f73c98126f7a03a7588ac0000000000000000266a24aa21a9ed96534de5e5fcddf2860d91401338b407be05094d29778ad9f3113c1a3d13ff9c00000000",["dd8452432335af3ed3fc0f05713a66e82e20371b0ee34b269f47155014d825c5","118a329a1d4455ddb98907045e5cbf7c5015c0d0f1239ff67eb44703c731cee1","f74f37f9ef87eaf474767bbf28cdd74f9af8698299f433b8b35368378578178b","cc74d73de09e0c405c1768efa3117cac67f6450d930ec773c37ba5881f43a36a","2b264b87d1c9c3632e583ca05af81f311d43ca2a0e2dee070965577bd84b8276","a5f69ee2f635c8a0a034fc1300ab588b8ab103d6c530e57a602d195c012d07d3","428b99f9860efc0c39873f4e654f5a08d74a0d880698526ace13906e30329236","ca64a7e8892ccba4145bbe40aeaecc35d4b02eb0fc1d90bc4e0fb60b6aa84c77","f98d67e0a826cdb1ad7c542922792c5bef43cfb10bea6725b139060d211fcb91","0ff1acba2d0a8ac624b0c2b1577a19d5344a06c4c528f895ebd72c28ebb69d58","7817abd43474b4dbf702167676020a2f34c79a31c6d285943bd70617f56bcbb7"],"20000000","170901ba","62817ca5",false]}
	{"id":6300,"method":"mining.submit","params":["seanmcadamworker0","6c84558d","e6b9000000000000","62817ca5","13666b8a"]}

	{"id":"5","jsonrpc":"2.0","method":"mining.notify","params":["b7f445ec","a1ac7997ce3af8f42a60e6982e378952456914d00004dc270000000000000000","01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff2103133d0b0004e17c81620c","5f546974616e2e696f5fffffffff0236d94226000000001976a914288913831abe556f331ed13f73c98126f7a03a7588ac0000000000000000266a24aa21a9ed050f8413de5f3275dcc6705c123af22dde01508831fc34c2936ca1549283250c00000000",["dd8452432335af3ed3fc0f05713a66e82e20371b0ee34b269f47155014d825c5","118a329a1d4455ddb98907045e5cbf7c5015c0d0f1239ff67eb44703c731cee1","f74f37f9ef87eaf474767bbf28cdd74f9af8698299f433b8b35368378578178b","cc74d73de09e0c405c1768efa3117cac67f6450d930ec773c37ba5881f43a36a","8d213def94c673bc8559a2aafc580d8791ccdd3200a6618569f49b7563a47bb4","6fbb06074fad7921afd284c7a6b721df6ce67996b500186382f6b56c3b8d8ca7","dd4b547d6acc835244010bba91cb071d39c12eb9f9fb21c1d8f881125ff0e29c","e8fef97dee58968910b7c16364238f8118031b0826a3aff7940b5455989f530c","06561b286dac7144c5f8f5c97b154572a2bfe15d70cdaba8485bbdce92156b9a","b62142c70208eb01b060446d58fd69f92b4812e085a5c35bca1e7fc95b126aa7","9dc2c631246a453e61c18422148fc4ac9bf627b09db1e95d3fcbf9e610b7849e"],"20000000","170901ba","62817ce1",false]} 
	{"id":6301,"method":"mining.submit","params":["seanmcadamworker0","b7f445ec","e78d000000000000","62817ce1","c80dafde"]}
	{"id":6302,"method":"mining.submit","params":["seanmcadamworker0","b7f445ec","7d33010000000000","62817ce1","94397a17"]}
	{"id":6303,"method":"mining.submit","params":["seanmcadamworker0","b7f445ec","bc42010000000000","62817ce1","46cdaded"]}

	{"id":"6","jsonrpc":"2.0","method":"mining.notify","params":["7862fc17","a1ac7997ce3af8f42a60e6982e378952456914d00004dc270000000000000000","01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff2103133d0b00041d7d81620c","5f546974616e2e696f5fffffffff02de1a4826000000001976a914288913831abe556f331ed13f73c98126f7a03a7588ac0000000000000000266a24aa21a9edbc839b3065fd92f12c559f1800244f5532afb0e11db84fbe3e8801ce372a48b900000000",["9533d69c1f998f31a4111892e31789587c7c730aff9b6df10f80a2085f84277e","542fc015cd88514b933d637290c84d37fbdad62373a27108f3a378656d9a302e","e90e5deb1bfc01549ba3df32620457203c167d637122274d1122fff144a268d4","2e62f2fbbc6114c31d6f7679a87863f91279d2979122a928f6c63282229811f1","a25f0657130532193f1b8f0f84ebf6e8574a3ee6532bacb254c1faff2f89e06b","ba8429ce07e2486b96f1466703940893c789c44b72daa340a4f97a7e30650542","98281d658ace8d4808164db9cd8fbf623160273c30b589e300d33eb5ba63f249","894683c961f32adaab68110fe8149a3b5c2c19bfd027d9a4a0cf579dacc7fa7d","c95e1f0ce91a93d41a9a094071e04007eb56bcda114e0ecc09b58032d0f88102","a6ae2ad41ced116c26cae2f7ff28c436b479dcdfb4159a7206d68cada960ac14","033730dc20535745dec00d6c9f631496b262f63460a5fecb1c26e9cf8d58b510"],"20000000","170901ba","62817d1d",false]}
	{"id":6304,"method":"mining.submit","params":["seanmcadamworker0","7862fc17","7b07010000000000","62817d1d","cff3c10d"]}
	{"id":6305,"method":"mining.submit","params":["seanmcadamworker0","7862fc17","d69b020000000000","62817d1d","1a5c4213"]}
	*/


	notifyJobIds := [3]string{"6c84558d","b7f445ec","7862fc17"}
	notifyPrevBlocks := [3]string{"a1ac7997ce3af8f42a60e6982e378952456914d00004dc270000000000000000","a1ac7997ce3af8f42a60e6982e378952456914d00004dc270000000000000000","a1ac7997ce3af8f42a60e6982e378952456914d00004dc270000000000000000"}
	notifyGen1s := [3]string{"01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff2103133d0b0004a57c81620c","01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff2103133d0b0004a57c81620c","01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff2103133d0b0004a57c81620c"}
	notifyGen2s := [3]string{"5f546974616e2e696f5fffffffff0247953d26000000001976a914288913831abe556f331ed13f73c98126f7a03a7588ac0000000000000000266a24aa21a9ed96534de5e5fcddf2860d91401338b407be05094d29778ad9f3113c1a3d13ff9c00000000", "5f546974616e2e696f5fffffffff0236d94226000000001976a914288913831abe556f331ed13f73c98126f7a03a7588ac0000000000000000266a24aa21a9ed050f8413de5f3275dcc6705c123af22dde01508831fc34c2936ca1549283250c00000000", "5f546974616e2e696f5fffffffff02de1a4826000000001976a914288913831abe556f331ed13f73c98126f7a03a7588ac0000000000000000266a24aa21a9edbc839b3065fd92f12c559f1800244f5532afb0e11db84fbe3e8801ce372a48b900000000"}
	notifyMerkless := [][]interface{}{{"dd8452432335af3ed3fc0f05713a66e82e20371b0ee34b269f47155014d825c5","118a329a1d4455ddb98907045e5cbf7c5015c0d0f1239ff67eb44703c731cee1","f74f37f9ef87eaf474767bbf28cdd74f9af8698299f433b8b35368378578178b","cc74d73de09e0c405c1768efa3117cac67f6450d930ec773c37ba5881f43a36a","2b264b87d1c9c3632e583ca05af81f311d43ca2a0e2dee070965577bd84b8276","a5f69ee2f635c8a0a034fc1300ab588b8ab103d6c530e57a602d195c012d07d3","428b99f9860efc0c39873f4e654f5a08d74a0d880698526ace13906e30329236","ca64a7e8892ccba4145bbe40aeaecc35d4b02eb0fc1d90bc4e0fb60b6aa84c77","f98d67e0a826cdb1ad7c542922792c5bef43cfb10bea6725b139060d211fcb91","0ff1acba2d0a8ac624b0c2b1577a19d5344a06c4c528f895ebd72c28ebb69d58","7817abd43474b4dbf702167676020a2f34c79a31c6d285943bd70617f56bcbb7"},
	{"dd8452432335af3ed3fc0f05713a66e82e20371b0ee34b269f47155014d825c5","118a329a1d4455ddb98907045e5cbf7c5015c0d0f1239ff67eb44703c731cee1","f74f37f9ef87eaf474767bbf28cdd74f9af8698299f433b8b35368378578178b","cc74d73de09e0c405c1768efa3117cac67f6450d930ec773c37ba5881f43a36a","8d213def94c673bc8559a2aafc580d8791ccdd3200a6618569f49b7563a47bb4","6fbb06074fad7921afd284c7a6b721df6ce67996b500186382f6b56c3b8d8ca7","dd4b547d6acc835244010bba91cb071d39c12eb9f9fb21c1d8f881125ff0e29c","e8fef97dee58968910b7c16364238f8118031b0826a3aff7940b5455989f530c","06561b286dac7144c5f8f5c97b154572a2bfe15d70cdaba8485bbdce92156b9a","b62142c70208eb01b060446d58fd69f92b4812e085a5c35bca1e7fc95b126aa7","9dc2c631246a453e61c18422148fc4ac9bf627b09db1e95d3fcbf9e610b7849e"},
	{"9533d69c1f998f31a4111892e31789587c7c730aff9b6df10f80a2085f84277e","542fc015cd88514b933d637290c84d37fbdad62373a27108f3a378656d9a302e","e90e5deb1bfc01549ba3df32620457203c167d637122274d1122fff144a268d4","2e62f2fbbc6114c31d6f7679a87863f91279d2979122a928f6c63282229811f1","a25f0657130532193f1b8f0f84ebf6e8574a3ee6532bacb254c1faff2f89e06b","ba8429ce07e2486b96f1466703940893c789c44b72daa340a4f97a7e30650542","98281d658ace8d4808164db9cd8fbf623160273c30b589e300d33eb5ba63f249","894683c961f32adaab68110fe8149a3b5c2c19bfd027d9a4a0cf579dacc7fa7d","c95e1f0ce91a93d41a9a094071e04007eb56bcda114e0ecc09b58032d0f88102","a6ae2ad41ced116c26cae2f7ff28c436b479dcdfb4159a7206d68cada960ac14","033730dc20535745dec00d6c9f631496b262f63460a5fecb1c26e9cf8d58b510"}}
	notifyVersions := [3]string{"20000000","20000000","20000000"}
	notifyNbitss := [3]string{"170901ba","170901ba","170901ba"}
	notifyNtimes := [3]string{"62817ca5" ,"62817ce1","62817d1d"}
	notifyCleans := [3]bool{false,false,false}

	workerNames := [6]string{"seanmcadamworker0","seanmcadamworker1","seanmcadamworker1","seanmcadamworker1","seanmcadamworker2","seanmcadamworker2"}
	jobIDs := [6]string{"6c84558d","b7f445ec","b7f445ec","b7f445ec","7862fc17","7862fc17"}
	extraNonce2s := [6]string{"e6b9000000000000","e78d000000000000","7d33010000000000","bc42010000000000","7b07010000000000","d69b020000000000"}
	nTimes := [6]string{"62817ca5","62817ce1","62817ce1","62817ce1","62817d1d","62817d1d"}
	nOnces := [6]string{"13666b8a","c80dafde","94397a17","46cdaded","cff3c10d","1a5c4213"}

	// Miner 1
	ps.SendValidateNotify(context.Background(), miner1.ID, defaultDestID, workerNames[0], notifyJobIds[0], notifyPrevBlocks[0], notifyGen1s[0], notifyGen2s[0], notifyMerkless[0], notifyVersions[0], notifyNbitss[0], notifyNtimes[0], notifyCleans[0])
	time.Sleep(time.Second * 3)

	ps.SendValidateSubmit(context.Background(), workerNames[0], miner1.ID, defaultDestID, jobIDs[0], extraNonce2s[0], nTimes[0], nOnces[0])
	time.Sleep(time.Second * 3)

	// Miner 2
	ps.SendValidateNotify(context.Background(), miner2.ID, defaultDestID, workerNames[1], notifyJobIds[1], notifyPrevBlocks[1], notifyGen1s[1], notifyGen2s[1], notifyMerkless[1], notifyVersions[1], notifyNbitss[1], notifyNtimes[1], notifyCleans[1])
	time.Sleep(time.Second * 3)

	ps.SendValidateSubmit(context.Background(), workerNames[1], miner2.ID, defaultDestID, jobIDs[1], extraNonce2s[1], nTimes[1], nOnces[1])
	time.Sleep(time.Second * 3)
	
	ps.SendValidateSubmit(context.Background(), workerNames[2], miner2.ID, defaultDestID, jobIDs[2], extraNonce2s[2], nTimes[2], nOnces[2])
	time.Sleep(time.Second * 3)

	ps.SendValidateSubmit(context.Background(), workerNames[3], miner2.ID, defaultDestID, jobIDs[3], extraNonce2s[3], nTimes[3], nOnces[3])
	time.Sleep(time.Second * 3)

	// Miner 3
	ps.SendValidateNotify(context.Background(), miner3.ID, defaultDestID, workerNames[4], notifyJobIds[2], notifyPrevBlocks[2], notifyGen1s[2], notifyGen2s[2], notifyMerkless[2], notifyVersions[2], notifyNbitss[2], notifyNtimes[2], notifyCleans[2])
	time.Sleep(time.Second * 3)

	ps.SendValidateSubmit(context.Background(), workerNames[4], miner3.ID, defaultDestID, jobIDs[4], extraNonce2s[4], nTimes[4], nOnces[4])
	time.Sleep(time.Second * 3)
	
	ps.SendValidateSubmit(context.Background(), workerNames[5], miner3.ID, defaultDestID, jobIDs[5], extraNonce2s[5], nTimes[5], nOnces[5])
	time.Sleep(time.Second * 3)

	m1,_ := ps.MinerGetWait(miner1.ID)
	m2,_ := ps.MinerGetWait(miner2.ID)
	m3,_ := ps.MinerGetWait(miner3.ID)

	fmt.Println("Miner 1 Current Hashrate: ", m1.CurrentHashRate)
	fmt.Println("Miner 2 Current Hashrate: ", m2.CurrentHashRate)
	fmt.Println("Miner 3 Current Hashrate: ", m3.CurrentHashRate)
	time.Sleep(time.Second * 2)

	fmt.Print("\n\n/// 2 New available contracts found ///\n\n\n")

	contract1 := msgbus.Contract{
		IsSeller: true,
		ID:       msgbus.ContractID("ContractID01"),
		State:    msgbus.ContAvailableState,
		Price:    10,
		Limit:    10,
		Speed:    2305843009213694000,
	}
	contract2 := msgbus.Contract{
		IsSeller: true,
		ID:       msgbus.ContractID("ContractID02"),
		State:    msgbus.ContAvailableState,
		Price:    10,
		Limit:    10,
		Speed:    2305843009213694000,
	}
	ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contract1.ID), contract1)
	ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contract2.ID), contract2)
	time.Sleep(time.Second * 2)
	
	fmt.Print("\n\n/// Contract 1 purchased and now running ///\n\n\n")

	targetDest := msgbus.Dest{
		ID:     msgbus.DestID(msgbus.GetRandomIDString()),
		NetUrl: "stratum+tcp://127.0.0.1:55555/",
	}
	ps.PubWait(msgbus.DestMsg, msgbus.IDString(targetDest.ID), targetDest)

	contract1.State = msgbus.ContRunningState
	contract1.Buyer = "buyer1"
	contract1.Dest = targetDest.ID
	ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contract1.ID), contract1)
	time.Sleep(reAdjustmentTime*time.Second)

	fmt.Print("\n--Time Slice 1--\n")
	minerIDs,_ := ps.MinerGetAllWait()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()

	time.Sleep((hashrateCalcLagTime/2 - reAdjustmentTime)*time.Second)

	time.Sleep(reAdjustmentTime*time.Second)

	fmt.Print("\n--Time Slice 2--\n")
	minerIDs,_ = ps.MinerGetAllWait()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()

	time.Sleep((hashrateCalcLagTime/2 - reAdjustmentTime)*time.Second)

	fmt.Print("\n\n/// Contract 2 purchased and now running ///\n\n\n")

	targetDest2 := msgbus.Dest{
		ID:     msgbus.DestID(msgbus.GetRandomIDString()),
		NetUrl: "stratum+tcp://127.0.0.1:66666/",
	}
	ps.PubWait(msgbus.DestMsg, msgbus.IDString(targetDest2.ID), targetDest2)

	contract2.State = msgbus.ContRunningState
	contract2.Buyer = "buyer2"
	contract2.Dest = targetDest2.ID
	ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contract2.ID), contract2)
	time.Sleep(hashrateCalcLagTime*time.Second)
	time.Sleep(reAdjustmentTime*time.Second)

	fmt.Print("\n--Time Slice 1--\n")
	minerIDs,_ = ps.MinerGetAllWait()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()

	time.Sleep((hashrateCalcLagTime/2 - reAdjustmentTime)*time.Second)

	time.Sleep(reAdjustmentTime*time.Second)

	fmt.Print("\n--Time Slice 2--\n")
	minerIDs,_ = ps.MinerGetAllWait()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()

	time.Sleep((hashrateCalcLagTime/2 - reAdjustmentTime)*time.Second)

	fmt.Print("\n\n/// Contract 1 closes out ///\n\n\n")

	contract1.State = msgbus.ContAvailableState
	ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contract1.ID), contract1)
	time.Sleep(hashrateCalcLagTime*time.Second)
	time.Sleep(reAdjustmentTime*time.Second)

	minerIDs,_ = ps.MinerGetAllWait()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}

	time.Sleep((hashrateCalcLagTime/2 - reAdjustmentTime)*time.Second)

	fmt.Print("\n\n/// Contract 2 closes out ///\n\n\n")

	contract2.State = msgbus.ContAvailableState
	ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contract2.ID), contract2)
	time.Sleep(hashrateCalcLagTime*time.Second)
	time.Sleep(reAdjustmentTime*time.Second)

	minerIDs,_ = ps.MinerGetAllWait()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
}