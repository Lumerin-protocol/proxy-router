package connectionscheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/connections"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

func TestPassthroughConnectionScheduler(t *testing.T) {
	l := log.New()
	ps := msgbus.New(10, l)

	ctxStruct := contextlib.NewContextStruct(nil, ps, nil, nil, nil)
	mainCtx := context.WithValue(context.Background(), contextlib.ContextKey, ctxStruct)

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

	nodeOperator := msgbus.NodeOperator{
		ID:          msgbus.NodeOperatorID(msgbus.GetRandomIDString()),
		DefaultDest: defaultDest.ID,
		IsBuyer:     false,
	}

	cs, err := New(&mainCtx, &nodeOperator, true, 2, connections.CreateConnectionCollection())
	if err != nil {
		panic(fmt.Sprintf("schedule manager failed:%s", err))
	}
	err = cs.Start()
	if err != nil {
		panic(fmt.Sprintf("schedule manager failed to start:%s", err))
	}

	fmt.Print("\n\n/// Multiple miners connecting to node ///\n\n\n")

	miner1 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID01"),
		IP:              "IpAddress1",
		CurrentHashRate: 27,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	miner2 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID02"),
		IP:              "IpAddress2",
		CurrentHashRate: 35,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	miner3 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID03"),
		IP:              "IpAddress3",
		CurrentHashRate: 72,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner1.ID), miner1)
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner2.ID), miner2)
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner3.ID), miner3)

	time.Sleep(time.Second * 2)

	fmt.Print("\n\n/// New available contract found ///\n\n\n")

	contract1 := msgbus.Contract{
		IsSeller: true,
		ID:       msgbus.ContractID("ContractID01"),
		State:    msgbus.ContAvailableState,
		Price:    10,
		Limit:    10,
		Speed:    100,
	}
	ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contract1.ID), contract1)

	time.Sleep(time.Second * 2)

	fmt.Print("\n\n/// Contract purchased and now running ///\n\n\n")

	targetDest := msgbus.Dest{
		ID:     msgbus.DestID(msgbus.GetRandomIDString()),
		NetUrl: "stratum+tcp://127.0.0.1:55555/",
	}
	ps.PubWait(msgbus.DestMsg, msgbus.IDString(targetDest.ID), targetDest)

	contract1.State = msgbus.ContRunningState
	contract1.Buyer = "buyer"
	contract1.Dest = targetDest.ID
	ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contract1.ID), contract1)
	time.Sleep(time.Second * 2)

	miners, _ := ps.MinerGetAllWait()

	for _, m := range miners {
		miner, _ := ps.MinerGetWait(m)
		if _,ok :=miner.Contracts[contract1.ID]; !ok || miner.Dest != contract1.Dest {
			t.Errorf("Miner contract and dest field incorrect")
		}
	}

	time.Sleep(time.Second * 2)

	fmt.Print("\n\n/// New miner connected ///\n\n\n")

	miner4 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID04"),
		IP:              "IpAddress4",
		CurrentHashRate: 88,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner4.ID), miner4)
	time.Sleep(time.Second * 5)

	miner, _ := ps.MinerGetWait(miner4.ID)
	if _,ok := miner.Contracts[contract1.ID]; !ok || miner.Dest != contract1.Dest {
		t.Errorf("Miner contract and dest field incorrect")
	}

	time.Sleep(time.Second * 2)

	fmt.Print("\n\n/// Contract closes out ///\n\n\n")

	contract1.State = msgbus.ContAvailableState
	ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contract1.ID), contract1)
	time.Sleep(time.Second * 5)

	miners, _ = ps.MinerGetAllWait()

	for _, m := range miners {
		miner, _ := ps.MinerGetWait(m)
		if len(miner.Contracts) != 0 || miner.Dest != nodeOperator.DefaultDest {
			t.Errorf("Miner contract and dest field incorrect")
		}
	}

	time.Sleep(time.Second * 2)
}

/*
On Ropsten, demonstrate one seller, three miners, two contracts (valued at 1+partial miner capacity),
purchased by two separate buyers concurrently and can show purchased hashrate consistently for the duration of both contracts
*/
func TestTimeSlicing(t *testing.T) {
	l := log.New()
	ps := msgbus.New(10, l)

	ctxStruct := contextlib.NewContextStruct(nil, ps, nil, nil, nil)
	mainCtx := context.WithValue(context.Background(), contextlib.ContextKey, ctxStruct)

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

	nodeOperator := msgbus.NodeOperator{
		ID:          msgbus.NodeOperatorID(msgbus.GetRandomIDString()),
		DefaultDest: defaultDest.ID,
		IsBuyer:     false,
	}

	var hashrateCalcLagTime time.Duration = 20
	var reAdjustmentTime time.Duration = 3

	cs, err := New(&mainCtx, &nodeOperator, false, int(hashrateCalcLagTime), connections.CreateConnectionCollection())
	if err != nil {
		panic(fmt.Sprintf("schedule manager failed:%s", err))
	}
	err = cs.Start()
	if err != nil {
		panic(fmt.Sprintf("schedule manager failed to start:%s", err))
	}

	fmt.Print("\n\n/// Multiple miners connecting to node ///\n\n\n")

	miner1 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID01"),
		IP:              "IpAddress1",
		CurrentHashRate: 0,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	miner2 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID02"),
		IP:              "IpAddress2",
		CurrentHashRate: 0,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	miner3 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID03"),
		IP:              "IpAddress3",
		CurrentHashRate: 0,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner1.ID), miner1)
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner2.ID), miner2)
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner3.ID), miner3)

	time.Sleep(time.Second * hashrateCalcLagTime)

	fmt.Print("\n\n/// Validator updated miner hashrates ///\n\n\n")

	miner1.CurrentHashRate = 100
	miner2.CurrentHashRate = 100
	miner3.CurrentHashRate = 100
	ps.SetWait(msgbus.MinerMsg, msgbus.IDString(miner1.ID), miner1)
	ps.SetWait(msgbus.MinerMsg, msgbus.IDString(miner2.ID), miner2)
	ps.SetWait(msgbus.MinerMsg, msgbus.IDString(miner3.ID), miner3)
	time.Sleep(time.Second * 1)

	fmt.Print("\n\n/// 2 New available contracts found ///\n\n\n")

	contract1 := msgbus.Contract{
		IsSeller: true,
		ID:       msgbus.ContractID("ContractID01"),
		State:    msgbus.ContAvailableState,
		Price:    10,
		Limit:    10,
		Speed:    150,
	}
	contract2 := msgbus.Contract{
		IsSeller: true,
		ID:       msgbus.ContractID("ContractID02"),
		State:    msgbus.ContAvailableState,
		Price:    10,
		Limit:    10,
		Speed:    150,
	}
	ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contract1.ID), contract1)
	ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contract2.ID), contract2)

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
	fmt.Println()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()

	time.Sleep((hashrateCalcLagTime/2 - reAdjustmentTime)*time.Second)

	fmt.Print("\n\n/// Contract 2 closes out ///\n\n\n")

	contract2.State = msgbus.ContAvailableState
	ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contract2.ID), contract2)
	time.Sleep(hashrateCalcLagTime*time.Second)
	time.Sleep(reAdjustmentTime*time.Second)

	minerIDs,_ = ps.MinerGetAllWait()
	fmt.Println()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()
}

func TestMultiTimeSlicing(t *testing.T) {
	l := log.New()
	ps := msgbus.New(10, l)

	ctxStruct := contextlib.NewContextStruct(nil, ps, nil, nil, nil)
	mainCtx := context.WithValue(context.Background(), contextlib.ContextKey, ctxStruct)

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

	nodeOperator := msgbus.NodeOperator{
		ID:          msgbus.NodeOperatorID(msgbus.GetRandomIDString()),
		DefaultDest: defaultDest.ID,
		IsBuyer:     false,
	}

	var hashrateCalcLagTime time.Duration = 20
	var reAdjustmentTime time.Duration = 3

	cs, err := New(&mainCtx, &nodeOperator, false, int(hashrateCalcLagTime), connections.CreateConnectionCollection())
	if err != nil {
		panic(fmt.Sprintf("schedule manager failed:%s", err))
	}
	err = cs.Start()
	if err != nil {
		panic(fmt.Sprintf("schedule manager failed to start:%s", err))
	}


	fmt.Print("\n\n/// Multiple miners connecting to node ///\n\n\n")

	miner1 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID01"),
		IP:              "IpAddress1",
		CurrentHashRate: 0,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	miner2 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID02"),
		IP:              "IpAddress2",
		CurrentHashRate: 0,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	miner3 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID03"),
		IP:              "IpAddress3",
		CurrentHashRate: 0,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner1.ID), miner1)
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner2.ID), miner2)
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner3.ID), miner3)

	time.Sleep(time.Second * hashrateCalcLagTime)


	fmt.Print("\n\n/// Validator updated miner hashrates ///\n\n\n")

	miner1.CurrentHashRate = 100
	miner2.CurrentHashRate = 100
	miner3.CurrentHashRate = 100
	ps.SetWait(msgbus.MinerMsg, msgbus.IDString(miner1.ID), miner1)
	ps.SetWait(msgbus.MinerMsg, msgbus.IDString(miner2.ID), miner2)
	ps.SetWait(msgbus.MinerMsg, msgbus.IDString(miner3.ID), miner3)
	time.Sleep(time.Second * 1)


	fmt.Print("\n\n/// 2 New available contracts found ///\n\n\n")

	contract1 := msgbus.Contract{
		IsSeller: true,
		ID:       msgbus.ContractID("ContractID01"),
		State:    msgbus.ContAvailableState,
		Price:    10,
		Limit:    10,
		Speed:    175,
	}
	contract2 := msgbus.Contract{
		IsSeller: true,
		ID:       msgbus.ContractID("ContractID02"),
		State:    msgbus.ContAvailableState,
		Price:    10,
		Limit:    10,
		Speed:    125,
	}
	ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contract1.ID), contract1)
	ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contract2.ID), contract2)


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

	time.Sleep((3*hashrateCalcLagTime/4 - reAdjustmentTime)*time.Second)

	time.Sleep(reAdjustmentTime*time.Second)

	fmt.Print("\n--Time Slice 2--\n")
	minerIDs,_ = ps.MinerGetAllWait()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()

	time.Sleep((hashrateCalcLagTime/4 - reAdjustmentTime)*time.Second)


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

	time.Sleep((3*hashrateCalcLagTime/4 - reAdjustmentTime)*time.Second)

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
	fmt.Println()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()

	time.Sleep((hashrateCalcLagTime/4 - reAdjustmentTime)*time.Second)


	fmt.Print("\n\n/// Contract 2 closes out ///\n\n\n")

	contract2.State = msgbus.ContAvailableState
	ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contract2.ID), contract2)
	time.Sleep(hashrateCalcLagTime*time.Second)
	time.Sleep(reAdjustmentTime*time.Second)

	minerIDs,_ = ps.MinerGetAllWait()
	fmt.Println()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()
}

func TestEdgeCases (t *testing.T) {
	l := log.New()
	ps := msgbus.New(10, l)

	ctxStruct := contextlib.NewContextStruct(nil, ps, nil, nil, nil)
	mainCtx := context.WithValue(context.Background(), contextlib.ContextKey, ctxStruct)

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

	nodeOperator := msgbus.NodeOperator{
		ID:          msgbus.NodeOperatorID(msgbus.GetRandomIDString()),
		DefaultDest: defaultDest.ID,
		IsBuyer:     false,
	}

	var hashrateCalcLagTime time.Duration = 20
	var reAdjustmentTime time.Duration = 3

	cs, err := New(&mainCtx, &nodeOperator, false, int(hashrateCalcLagTime), connections.CreateConnectionCollection())
	if err != nil {
		panic(fmt.Sprintf("schedule manager failed:%s", err))
	}
	err = cs.Start()
	if err != nil {
		panic(fmt.Sprintf("schedule manager failed to start:%s", err))
	}


	fmt.Print("\n\n/// Multiple miners connecting to node ///\n\n\n")

	miner1 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID01"),
		IP:              "IpAddress1",
		CurrentHashRate: 0,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	miner2 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID02"),
		IP:              "IpAddress2",
		CurrentHashRate: 0,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	miner3 := msgbus.Miner{
		ID:              msgbus.MinerID("MinerID03"),
		IP:              "IpAddress3",
		CurrentHashRate: 0,
		State:           msgbus.OnlineState,
		Dest:            defaultDest.ID,
		Contracts:       make(map[msgbus.ContractID]float64),
	}
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner1.ID), miner1)
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner2.ID), miner2)
	ps.PubWait(msgbus.MinerMsg, msgbus.IDString(miner3.ID), miner3)

	time.Sleep(time.Second * hashrateCalcLagTime)


	fmt.Print("\n\n/// Validator updated miner hashrates ///\n\n\n")

	miner1.CurrentHashRate = 200
	miner2.CurrentHashRate = 140
	miner3.CurrentHashRate = 140
	ps.SetWait(msgbus.MinerMsg, msgbus.IDString(miner1.ID), miner1)
	ps.SetWait(msgbus.MinerMsg, msgbus.IDString(miner2.ID), miner2)
	ps.SetWait(msgbus.MinerMsg, msgbus.IDString(miner3.ID), miner3)
	time.Sleep(time.Second * 1)


	fmt.Print("\n\n/// 2 New available contracts found ///\n\n\n")

	contract1 := msgbus.Contract{
		IsSeller: true,
		ID:       msgbus.ContractID("ContractID01"),
		State:    msgbus.ContAvailableState,
		Price:    10,
		Limit:    10,
		Speed:    270,
	}
	contract2 := msgbus.Contract{
		IsSeller: true,
		ID:       msgbus.ContractID("ContractID02"),
		State:    msgbus.ContAvailableState,
		Price:    10,
		Limit:    10,
		Speed:    110,
	}
	ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contract1.ID), contract1)
	ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contract2.ID), contract2)


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

	time.Sleep((3*hashrateCalcLagTime/4 - reAdjustmentTime)*time.Second)

	time.Sleep(reAdjustmentTime*time.Second)

	fmt.Print("\n--Time Slice 2--\n")
	minerIDs,_ = ps.MinerGetAllWait()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()

	time.Sleep((hashrateCalcLagTime/4 - reAdjustmentTime)*time.Second)


	fmt.Print("\n\n/// Miner disconnects and reconnects ///\n\n\n")

	time.Sleep(reAdjustmentTime*time.Second)

	ps.UnpubWait(msgbus.MinerMsg, msgbus.IDString(miner3.ID))
	time.Sleep(reAdjustmentTime*time.Second)
	
	miner3.ID = msgbus.MinerID("MinerID03-Reconnect")
	time.Sleep(reAdjustmentTime*time.Second)

	fmt.Print("\n--Time Slice 1--\n")
	minerIDs,_ = ps.MinerGetAllWait()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()

	time.Sleep((3*hashrateCalcLagTime/4 - reAdjustmentTime)*time.Second)

	time.Sleep(reAdjustmentTime*time.Second)

	fmt.Print("\n--Time Slice 2--\n")
	minerIDs,_ = ps.MinerGetAllWait()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()

	time.Sleep((hashrateCalcLagTime/4 - reAdjustmentTime)*time.Second)


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

	time.Sleep((3*hashrateCalcLagTime/4 - reAdjustmentTime)*time.Second)

	time.Sleep(reAdjustmentTime*time.Second)

	fmt.Print("\n--Time Slice 2--\n")
	minerIDs,_ = ps.MinerGetAllWait()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()

	time.Sleep((hashrateCalcLagTime/2 - reAdjustmentTime)*time.Second)


	fmt.Print("\n\n/// Contract 1 closes out in middle of slice period ///\n\n\n")

	time.Sleep(reAdjustmentTime*time.Second)
	contract1.State = msgbus.ContAvailableState
	ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contract1.ID), contract1)
	time.Sleep(hashrateCalcLagTime*time.Second)
	time.Sleep(reAdjustmentTime*time.Second)

	minerIDs,_ = ps.MinerGetAllWait()
	fmt.Println()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()

	time.Sleep((hashrateCalcLagTime/4 - reAdjustmentTime)*time.Second)


	fmt.Print("\n\n/// Contract 2 closes out ///\n\n\n")

	contract2.State = msgbus.ContAvailableState
	ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contract2.ID), contract2)
	time.Sleep(hashrateCalcLagTime*time.Second)
	time.Sleep(reAdjustmentTime*time.Second)

	minerIDs,_ = ps.MinerGetAllWait()
	fmt.Println()
	for i,m := range minerIDs {
		miner,_ := ps.MinerGetWait(m)
		fmt.Printf("Miner%d: %v\n", i, miner)
	}
	fmt.Println()
}