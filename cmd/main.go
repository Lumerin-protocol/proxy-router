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

	configs := config.ReadConfigFile()
	l.SetLevel(log.Level(configs.LogLevel))

	logFile, err := os.OpenFile(configs.LogFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		l.Logf(log.LevelError, "error opening log file: %v", err)
		return
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
	// the proto argument (#1) gets set in the Protocol sus-system
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
		l.Logf(log.LevelError, "Adding Default Dest Failed: %s", err)
		return
	}
	if event.Err != nil {
		l.Logf(log.LevelError, "Adding Default Dest Failed: %s", event.Err)
		return
	}

	//
	// Setup Node Operator Msg in msgbus
	//
	nodeOperator := msgbus.NodeOperator{
		ID:          msgbus.NodeOperatorID(msgbus.GetRandomIDString()),
		IsBuyer:     configs.BuyerNode,
		DefaultDest: dest.ID,
		Contracts: make(map[msgbus.ContractID]msgbus.ContractState),
	}
	event, err = ps.PubWait(msgbus.NodeOperatorMsg, msgbus.IDString(nodeOperator.ID), nodeOperator)
	if err != nil {
		l.Logf(log.LevelError, "Adding Node Operator Failed: %s", err)
		return
	}
	if event.Err != nil {
		l.Logf(log.LevelError, "Adding Node Operator Failed: %s", event.Err)
		return
	}

	//
	// Fire up the StratumV1 Potocol
	//
	if !configs.DisableStratumv1 {

		listenAddress := fmt.Sprintf("%s:%s", configs.ListenIP, configs.ListenPort)

		src, err := net.ResolveTCPAddr("tcp", listenAddress)
		if err != nil {
			l.Logf(log.LevelError, "Unable to resolve TCP Addr: %s", listenAddress)
			return
		}

		l.Logf(log.LevelInfo, "Listening for stratum messages on %v\n\n", src.String())

		stratum, err := stratumv1.NewListener(mainContext, src, dest)
		if err != nil {
			l.Logf(log.LevelError, "NewListener Error: %s", err)
			return
		}

		switchMethod := configs.SwitchMethod
		switchMethod = strings.ToLower(switchMethod)

		switch switchMethod {
		case "ondemand":
			stratum.SetScheduler(stratumv1.OnDemand)
		case "onsubmit":
			stratum.SetScheduler(stratumv1.OnSubmit)
		default:
			l.Logf(log.LevelError, "Scheduler value: %s Not Supported", switchMethod)
			return
		}

		serialize := configs.Serialize
		stratum.SetSerialize(serialize)

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
		var contractManagerConfig lumerinlib.ContractManagerConfig

		contractManagerConfig.Mnemonic = configs.Mnemonic
		contractManagerConfig.AccountIndex = configs.AccountIndex
		contractManagerConfig.EthNodeAddr = configs.EthNodeAddr
		contractManagerConfig.ClaimFunds = configs.ClaimFunds
		contractManagerConfig.TimeThreshold = configs.TimeThreshold
		contractManagerConfig.CloneFactoryAddress = configs.CloneFactoryAddress
		contractManagerConfig.LumerinTokenAddress = configs.LumerinTokenAddress
		contractManagerConfig.ValidatorAddress = configs.ValidatorAddress
		contractManagerConfig.ProxyAddress = configs.ProxyAddress

		if configs.BuyerNode {
			var buyerCM contractmanager.BuyerContractManager
			err = contractmanager.Run(&mainContext, &buyerCM, contractManagerConfig, &nodeOperator)
		} else {
			var sellerCM contractmanager.SellerContractManager
			err = contractmanager.Run(&mainContext, &sellerCM, contractManagerConfig, &nodeOperator)
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
		go api.Run(mainContext, configs.ApiPort)
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
