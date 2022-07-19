package config

import (
	"fmt"
	"strconv"
)

type ConfigRead struct {
	BuyerNode           bool
	DisableConnection   bool
	DisableStratumv1    bool
	ListenIP            string
	ListenPort          string
	DefaultPoolAddr     string
	DisableSchedule     bool
	SchedulePassthrough bool
	HashrateCalcLagTime int
	DisableValidate     bool
	DisableContract     bool
	Mnemonic            string
	AccountIndex        int
	EthNodeAddr         string
	ClaimFunds          bool
	TimeThreshold       int
	CloneFactoryAddress string
	LumerinTokenAddress string
	ValidatorAddress    string
	ProxyAddress        string
	DisableApi          bool
	ApiPort             string
	LogLevel            int
	LogFilePath         string
	SwitchMethod        string
	Serialize           bool
}

func ReadConfigs() (configs ConfigRead) {
	configFile, err := ConfigGetVal(ConfigConfigFilePath)
	if err != nil {
		panic(fmt.Sprintf("Getting Config File val failed: %s\n", err))
	}

	if configFile != "" { // if a config file was specified use it instead of flag params

		var ok bool
		//
		// Config Configs
		//
		configConfig, err := LoadConfiguration("config")
		if err != nil {
			panic(fmt.Sprintf("Failed to load config configuration: %v", err))
		}
		configs.BuyerNode = configConfig["buyerNode"].(bool)

		//
		// Connection Configs
		//
		connectionConfig, err := LoadConfiguration("connection")
		if err != nil {
			panic(fmt.Sprintf("Failed to load connection configuration: %v", err))
		}

		configs.DisableConnection, ok = connectionConfig["disable"].(bool)
		if !ok {
			configs.DisableConnection = false
		}

		configs.DisableStratumv1, ok = connectionConfig["disableStratumv1"].(bool)
		if !ok {
			configs.DisableConnection = false
		}

		configs.ListenIP, ok = connectionConfig["listenIP"].(string)
		if !ok {
			panic(fmt.Sprintf("Failed to load Connection Listen IP Address"))
		}

		configs.ListenPort, ok = connectionConfig["listenPort"].(string)
		if !ok {
			panic(fmt.Sprintf("Failed to load Connection Listen Port"))
		}

		configs.DefaultPoolAddr, ok = connectionConfig["defaultPoolAddr"].(string)
		if !ok {
			panic(fmt.Sprintf("Failed to load Connection Default Pool Address"))
		}

		configs.SwitchMethod, ok = connectionConfig["switchmethod"].(string)
		if !ok {
			configs.SwitchMethod = ConfigMap[ConfigConnectionSwitchMethod].defval
		}
		if !(configs.SwitchMethod == "ondemand" || configs.SwitchMethod == "onsubmit") {
			panic(fmt.Sprintf("Switch Method '%s' not supported", configs.SwitchMethod))
		}

		configs.Serialize, ok = connectionConfig["serializeworker"].(bool)
		if !ok {
			if ConfigMap[ConfigConnectionSerializeWorker].defval == "true" {
				configs.Serialize = true
			} else {
				configs.Serialize = false
			}
		}

		//
		// Scheduler Configs
		//
		scheduleConfig, err := LoadConfiguration("schedule")
		if err != nil {
			panic(fmt.Sprintf("Failed to load schedule configuration: %v", err))
		}
		configs.DisableSchedule = scheduleConfig["disable"].(bool)
		configs.SchedulePassthrough = scheduleConfig["passthrough"].(bool)
		configs.HashrateCalcLagTime = int(scheduleConfig["hashrateCalcLagTime"].(float64))

		//
		// Validate Configs
		//
		validateConfig, err := LoadConfiguration("validate")
		if err != nil {
			panic(fmt.Sprintf("Failed to load validate configuration: %v", err))
		}
		configs.DisableValidate = validateConfig["disable"].(bool)

		//
		// Contract Configs
		//
		contractConfig, err := LoadConfiguration("contract")
		if err != nil {
			panic(fmt.Sprintf("Failed to load contract configuration: %v", err))
		}

		configs.DisableContract = contractConfig["disable"].(bool)
		configs.Mnemonic = contractConfig["mnemonic"].(string)
		configs.AccountIndex = int(contractConfig["accountIndex"].(float64))
		configs.EthNodeAddr = contractConfig["ethNodeAddr"].(string)
		configs.ClaimFunds = contractConfig["claimFunds"].(bool)
		configs.TimeThreshold = int(contractConfig["timeThreshold"].(float64))
		configs.CloneFactoryAddress = contractConfig["cloneFactoryAddress"].(string)
		configs.LumerinTokenAddress = contractConfig["lumerinTokenAddress"].(string)
		configs.ValidatorAddress = contractConfig["validatorAddress"].(string)

		//
		// API Configs
		//
		apiConfig, err := LoadConfiguration("api")
		if err != nil {
			panic(fmt.Sprintf("Failed to load connection configuration: %v", err))
		}
		configs.DisableApi = apiConfig["disable"].(bool)
		configs.ApiPort = apiConfig["port"].(string)

		//
		// Logging Configs
		//
		loggingConfig, err := LoadConfiguration("logging")
		if err != nil {
			panic(fmt.Sprintf("Failed to load connection configuration: %v", err))
		}
		configs.LogLevel = int(loggingConfig["level"].(float64))
		configs.LogFilePath = loggingConfig["filePath"].(string)

		//
		// Load Defaults
		//
	} else {
		//
		// Config Configs
		//
		configs.BuyerNode = false
		buyerNodeStr, err := ConfigGetVal(BuyerNode)
		if err != nil {
			panic(fmt.Sprintf("Getting Buyer Node val failed: %s\n", err))
		}
		if buyerNodeStr == "true" {
			configs.BuyerNode = true
		}

		//
		// Connection Configs
		//

		configs.DisableConnection = false
		configs.DisableStratumv1 = false
		configs.SwitchMethod = "OnDemand"

		disableConnectionStr, err := ConfigGetVal(DisableConnection)
		if err == nil && disableConnectionStr == "true" {
			configs.DisableConnection = true
		}
		disableStratumv1Str, err := ConfigGetVal(DisableStratumv1)
		if err == nil && disableStratumv1Str == "true" {
			configs.DisableStratumv1 = true
		}

		configs.ListenIP, err = ConfigGetVal(ConfigConnectionListenIP)
		if err != nil {
			panic(fmt.Sprintf("Getting Listen IP val failed: %s\n", err))
		}
		configs.ListenPort, err = ConfigGetVal(ConfigConnectionListenPort)
		if err != nil {
			panic(fmt.Sprintf("Getting Listen Port val failed: %s\n", err))
		}
		configs.DefaultPoolAddr, err = ConfigGetVal(DefaultPoolAddr)
		if err != nil {
			panic(fmt.Sprintf("Getting Default Pool Addr val failed: %s\n", err))
		}

		switchMethodStr, err := ConfigGetVal(ConfigConnectionSwitchMethod)
		if err == nil && switchMethodStr == "onsubmit" {
			configs.SwitchMethod = "OnSubmit"
		}

		enableSerializeWorker, err := ConfigGetVal(ConfigConnectionSerializeWorker)
		if err == nil && enableSerializeWorker == "true" {
			configs.Serialize = true
		}

		//
		// Scheduler Configs
		//
		configs.DisableSchedule = false
		configs.SchedulePassthrough = false
		disableScheduleStr, err := ConfigGetVal(DisableSchedule)
		if err != nil {
			panic(fmt.Sprintf("Getting Disable Schedule val failed: %s\n", err))
		}
		if disableScheduleStr == "true" {
			configs.DisableSchedule = true
		}
		passthroughStr, err := ConfigGetVal(ConfigSchedulePassthrough)
		if err != nil {
			panic(fmt.Sprintf("Getting Schedule Passthrough val failed: %s\n", err))
		}
		if passthroughStr == "true" {
			configs.SchedulePassthrough = true
		}

		//
		// Validate Configs
		//
		configs.DisableValidate = false
		disableValidateStr, err := ConfigGetVal(DisableValidate)
		if err != nil {
			panic(fmt.Sprintf("Getting Disable Validate val failed: %s\n", err))
		}
		if disableValidateStr == "true" {
			configs.DisableValidate = true
		}

		//
		// Contract Configs
		//
		configs.DisableContract = false
		configs.ClaimFunds = false

		disableContractStr, err := ConfigGetVal(DisableContract)
		if err != nil {
			panic(fmt.Sprintf("Getting Disable Contract val failed: %s\n", err))
		}
		if disableContractStr == "true" {
			configs.DisableContract = true
		}
		configs.Mnemonic, err = ConfigGetVal(ConfigContractMnemonic)
		if err != nil {
			panic(fmt.Sprintf("Getting Mnemonic val failed: %s\n", err))
		}
		accountIndexStr, err := ConfigGetVal(ConfigContractAccountIndex)
		if err != nil {
			panic(fmt.Sprintf("Getting Account Index val failed: %s\n", err))
		}
		configs.AccountIndex, err = strconv.Atoi(accountIndexStr)
		if err != nil {
			panic(fmt.Sprintf("Converting Account Index string to int failed: %s\n", err))
		}
		configs.EthNodeAddr, err = ConfigGetVal(ConfigContractEthereumNodeAddress)
		if err != nil {
			panic(fmt.Sprintf("Getting Ethereum Node Address val failed: %s\n", err))
		}
		claimFundsStr, err := ConfigGetVal(ConfigContractClaimFunds)
		if err != nil {
			panic(fmt.Sprintf("Getting Claim Funds val failed: %s\n", err))
		}
		if claimFundsStr == "true" {
			configs.ClaimFunds = true
		}
		timeThresholdStr, err := ConfigGetVal(ConfigContractTimeThreshold)
		if err != nil {
			panic(fmt.Sprintf("Getting Time Threshold val failed: %s\n", err))
		}
		configs.TimeThreshold, err = strconv.Atoi(timeThresholdStr)
		if err != nil {
			panic(fmt.Sprintf("Converting Time Threshold string to int failed: %s\n", err))
		}

		network, err := ConfigGetVal(ConfigContractNetwork)
		if err != nil {
			panic(fmt.Sprintf("Getting Network val failed: %s\n", err))
		}
		switch network {
		case "ropsten":
			configs.CloneFactoryAddress = "0xe91be01493f4ae28297790277303926aaec604dc"
			configs.LumerinTokenAddress = "0xC6a30Bc2e1D7D9e9FFa5b45a21b6bDCBc109aE1B"
			configs.ValidatorAddress = "0x508CD3988E2b4B8f1d243b961a855347349f6F63"
		case "custom":
			configs.CloneFactoryAddress = "0xF735F5cFBC65EDcc67FE2F3f34413B3a66bA42E5"
			configs.LumerinTokenAddress = "0xf84D04A844D9a6F44c7F5bCd01b0852F47631c4e"
			configs.ValidatorAddress = "0x508CD3988E2b4B8f1d243b961a855347349f6F63"
		case "mainnet":
			configs.CloneFactoryAddress = "0xe91be01493f4ae28297790277303926aaec604dc"
			configs.LumerinTokenAddress = "0xC6a30Bc2e1D7D9e9FFa5b45a21b6bDCBc109aE1B"
			configs.ValidatorAddress = "0x508CD3988E2b4B8f1d243b961a855347349f6F63"
		default:
			panic(fmt.Sprintln("Invalid network input (must be ropsten, custom, or mainnet)"))
		}

		//
		// API Configs
		//
		configs.DisableApi = false
		disableApiStr, err := ConfigGetVal(DisableAPI)
		if err != nil {
			panic(fmt.Sprintf("Getting Disable API val failed: %s\n", err))
		}
		if disableApiStr == "true" {
			configs.DisableApi = true
		}
		configs.ApiPort, err = ConfigGetVal(ConfigRESTPort)
		if err != nil {
			panic(fmt.Sprintf("Getting API Port val failed: %s\n", err))
		}

		//
		// Logging Configs
		//
		logLevelStr, err := ConfigGetVal(ConfigLogLevel)
		if err != nil {
			panic(fmt.Sprintf("Getting Log Level val failed: %s\n", err))
		}
		configs.LogLevel, err = strconv.Atoi(logLevelStr)
		if err != nil {
			panic(fmt.Sprintf("Converting Log Level string to int failed: %s\n", err))
		}
		configs.LogFilePath, err = ConfigGetVal(ConfigLogFilePath)
		if err != nil {
			panic(fmt.Sprintf("Getting Log File Path val failed: %s\n", err))
		}
	}

	return configs
}
