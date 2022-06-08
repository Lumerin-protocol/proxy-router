package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"

	"gitlab.com/TitanInd/lumerin/cmd/config"
	"gitlab.com/TitanInd/lumerin/cmd/connectionscheduler"
	"gitlab.com/TitanInd/lumerin/cmd/contractmanager"
	"gitlab.com/TitanInd/lumerin/cmd/externalapi"
	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/protocol/stratumv1"
	"gitlab.com/TitanInd/lumerin/cmd/validator/validator"
	"gitlab.com/TitanInd/lumerin/connections"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

// -------------------------------------------
//
// Start up the modules one by one
// Config
// Log
// MsgBus
// Connection Manager
// Scheduling Manager
// Contract Manager
// External API
//
// -------------------------------------------

func main() {
	l := log.New()

	configs := config.ReadConfigs()
	l.SetLevel(log.Level(configs.LogLevel))

	logFile, err := os.OpenFile(configs.LogFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		l.Logf(log.LevelFatal, "error opening log file: %v", err)
	}
	defer logFile.Close()

	//l.SetFormat(log.FormatJSON).SetOutput(logFile)

	mainContext, mainCancel := context.WithCancel(context.Background())
	sigInt := make(chan os.Signal, 1)
	signal.Notify(sigInt, os.Interrupt)

	//
	// Fire up the Message Bus
	//
	ps := msgbus.New(10, l)

	//
	// Create Connection Collection
	//
	connectionCollection := connections.CreateConnectionCollection()

	// Add the various Context variables here
	// msgbus, logger, default listen address, defalt desitnation address
	//
	src := lumerinlib.NewNetAddr(lumerinlib.TCP, configs.ListenIP+":"+configs.ListenPort)
	dst := lumerinlib.NewNetAddr(lumerinlib.TCP, configs.DefaultPoolAddr)

	//
	// the proro argument (#1) gets set in the Protocol sus-system
	//
	cs := contextlib.NewContextStruct(nil, ps, l, src, dst)

	//
	// All of the various needed subsystem values get passed into the context here.
	//
	mainContext = context.WithValue(mainContext, contextlib.ContextKey, cs)

	//
	// Setup Default Dest in msgbus
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
	// Setup Node Operator Msg in msgbus
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
	if !configs.DisableStratumv1 {

		listenAddress := fmt.Sprintf("%s:%s", configs.ListenIP, configs.ListenPort)

		src, err := net.ResolveTCPAddr("tcp", listenAddress)
		if err != nil {
			lumerinlib.PanicHere("")
		}

		l.Logf(log.LevelInfo, "Listening for stratum messages on %v\n\n", src.String())

		stratum, err := stratumv1.NewListener(mainContext, src, dest)
		scheduler := configs.Scheduler
		scheduler = strings.ToLower(scheduler)

		switch scheduler {
		case "ondemand":
			stratum.SetScheduler(stratumv1.OnDemand)
		case "onsubmit":
			stratum.SetScheduler(stratumv1.OnSubmit)
		default:
			l.Logf(log.LevelPanic, "Scheduler value: %s Not Supported", scheduler)
		}

		if err != nil {
			panic(fmt.Sprintf("Stratum Protocol New() failed:%s", err))
		}

		stratum.Run()

	}

	//
	// Fire up schedule manager
	//
	if !configs.DisableSchedule {
		cs, err := connectionscheduler.New(&mainContext, &nodeOperator, configs.SchedulePassthrough, configs.HashrateCalcLagTime, connectionCollection)
		if err != nil {
			l.Logf(log.LevelPanic, "Schedule manager failed: %v", err)
		}
		err = cs.Start()
		if err != nil {
			l.Logf(log.LevelPanic, "Schedule manager failed to start: %v", err)
		}
	}

	//
	// Fire up validator
	//
	if !configs.DisableValidate {
		v := validator.MakeNewValidator(&mainContext)
		err = v.Start()
		if err != nil {
			l.Logf(log.LevelPanic, "Validator failed to start: %v", err)
		}
	}

	//
	// Fire up contract manager
	//
	if !configs.DisableContract {
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

		if configs.BuyerNode {
			var buyerCM contractmanager.BuyerContractManager
			err = contractmanager.Run(&mainContext, &buyerCM, msgbus.IDString(contractManagerConfig.ID), &nodeOperator)
		} else {
			var sellerCM contractmanager.SellerContractManager
			err = contractmanager.Run(&mainContext, &sellerCM, msgbus.IDString(contractManagerConfig.ID), &nodeOperator)
		}
		if err != nil {
			l.Logf(log.LevelPanic, "Contract manager failed to run: %v", err)
		}
	}

	//
	//Fire up external api
	//
	if !configs.DisableApi {
		api := externalapi.New(ps, connectionCollection)
		go api.Run(configs.ApiPort, l)
	}

	select {
	case <-sigInt:
		l.Logf(log.LevelWarn, "Signal Interupt: Cancelling all contexts and shuting down program")
		mainCancel()
	case <-mainContext.Done():
		time.Sleep(time.Second * 5)
		signal.Stop(sigInt)
		return
	}
}
