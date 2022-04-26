package config

//
// Define all configuration variables here
//

type ConfigConst string

const (
	BuyerNode                         ConfigConst = "BuyerNode"
	ConfigHelp                        ConfigConst = "ConfigHelp"
	ConfigContractNetwork             ConfigConst = "ConfigContractNetwork"
	ConfigContractMnemonic            ConfigConst = "ConfigContractMnemonic"
	ConfigContractEthereumNodeAddress ConfigConst = "ConfigContractEthereumNodeAddress"
	ConfigContractClaimFunds          ConfigConst = "ConfigContractClaimFunds"
	ConfigContractAccountIndex        ConfigConst = "ConfigContractAccountIndex"
	ConfigContractTimeThreshold       ConfigConst = "ConfigContractTimeThreshold"
	ConfigConnectionListenIP          ConfigConst = "ConfigConnectionListenIP"
	ConfigConnectionListenPort        ConfigConst = "ConfigConnectionListenPort"
	ConfigConfigFilePath              ConfigConst = "ConfigConfigFilePath"
	ConfigConfigDownloadPath          ConfigConst = "ConfigConfigDownloadPath"
	ConfigLogFilePath                 ConfigConst = "ConfigLogFilePath"
	ConfigLogLevel                    ConfigConst = "ConfigLogLevel"
	ConfigSchedulePassthrough         ConfigConst = "ConfigSchedulePassthrough"
	DefaultPoolAddr                   ConfigConst = "DefaultPoolAddr"
	ConfigRESTPort                    ConfigConst = "ConfigRESTPort"
	DisableConnection                 ConfigConst = "DisableConnection"
	DisableContract                   ConfigConst = "DisableContract"
	DisableSchedule                   ConfigConst = "DisableSchedule"
	DisableStratumv1                  ConfigConst = "DisableStratumV1"
	DisableAPI                        ConfigConst = "DisableAPI"
)

// Config Structure
type configitem struct {
	flagname   string
	flagusage  string
	envname    string
	configname string
	defval     string
	configval  *string
	envval     *string
	flagval    *string
}

//
// Define all Configuration constants that can be read in from the command line and the environment
//
var ConfigMap = map[ConfigConst]configitem{
	BuyerNode: {
		flagname:   "buyer",
		flagusage:  "Sets the system Seller or Buyer mode",
		envname:    "",
		configname: "",
		defval:     "false",
		configval:  nil,
		envval:     nil,
		flagval:    nil,
	},
	ConfigHelp: {
		flagname:   "help",
		flagusage:  "Display The help Screen",
		envname:    "",
		configname: "",
		defval:     "",
		configval:  nil,
		envval:     nil,
		flagval:    nil,
	},
	ConfigConnectionListenIP: {
		flagname:   "listenip",
		flagusage:  "IP to listen on",
		envname:    "LISTENIP",
		configname: "connect.listenip",
		defval:     "127.0.0.1",
		configval:  nil,
		envval:     nil,
		flagval:    nil,
	},
	ConfigConnectionListenPort: {
		flagname:   "listenport",
		flagusage:  "Connection Port to listen on",
		envname:    "LISTENPORT",
		configname: "connect.listenport",
		defval:     "3333",
		configval:  nil,
		envval:     nil,
		flagval:    nil,
	},
	ConfigContractNetwork: {
		flagname:   "network",
		flagusage:  "Options: mainnet, ropsten, or custom",
		envname:    "NEWTORK",
		configname: "contract.network",
		defval:     "custom",
		configval:  nil,
		envval:     nil,
		flagval:    nil,
	},
	ConfigContractMnemonic: {
		flagname:   "mnemonic",
		flagusage:  "HD Wallet Mnemonic",
		envname:    "MNEMONIC",
		configname: "contract.mnemonic",
		defval:     "",
		configval:  nil,
		envval:     nil,
		flagval:    nil,
	},
	ConfigContractEthereumNodeAddress: {
		flagname:   "ethnodeaddress",
		flagusage:  "URL of Ethereum Node",
		envname:    "ETHNODEADDRESS",
		configname: "contract.ethnode",
		defval:     "wss://127.0.0.1:7545",
		configval:  nil,
		envval:     nil,
		flagval:    nil,
	},
	ConfigContractClaimFunds: {
		flagname:   "claimfunds",
		flagusage:  "Seller Claims Funds at Closeout",
		envname:    "CLAIMFUNDS",
		configname: "contract.claimfunds",
		defval:     "false",
		configval:  nil,
		envval:     nil,
		flagval:    nil,
	},
	ConfigContractAccountIndex: {
		flagname:   "accountindex",
		flagusage:  "Account number in HD Wallet",
		envname:    "ACCOUNTINDEX",
		configname: "contract.accountindex",
		defval:     "0",
		configval:  nil,
		envval:     nil,
		flagval:    nil,
	},
	ConfigContractTimeThreshold: {
		flagname:   "timethreshold",
		flagusage:  "Time in seconds buyer contract manager's closeout monitor waits for miner hashrate to update before checking for closeout",
		envname:    "TIMETHRESHOLD",
		configname: "contract.timethreshold",
		defval:     "10",
		configval:  nil,
		envval:     nil,
		flagval:    nil,
	},
	ConfigConfigFilePath: {
		flagname:  "configfile",
		flagusage: "Relative Path to Configuration File",
		envname:   "CONFIGFILEPATH",
		defval:    "",
		configval: nil,
		envval:    nil,
		flagval:   nil,
	},
	ConfigConfigDownloadPath: {
		flagname:  "configdownload",
		flagusage: "Configuration Download Path",
		envname:   "CONFIGDOWNLOADURL",
		defval:    "",
		configval: nil,
		envval:    nil,
		flagval:   nil,
	},
	ConfigLogLevel: {
		flagname:  "loglevel",
		flagusage: "Logging level",
		envname:   "LOGLEVEL",
		defval:    "",
		configval: nil,
		envval:    nil,
		flagval:   nil,
	},
	ConfigLogFilePath: {
		flagname:  "logfile",
		flagusage: "Log File Path",
		envname:   "LOGFILEPATH",
		defval:    "lumerin.log",
		configval: nil,
		envval:    nil,
		flagval:   nil,
	},
	ConfigSchedulePassthrough: {
		flagname:  "schedulepassthrough",
		flagusage: "Run connection scheduler in passthrough mode i.e. no miner orchestration and only 1 contract running at a time",
		envname:   "SCHEDULEPASSTHROUGH",
		defval:    "false",
		configval: nil,
		envval:    nil,
		flagval:   nil,
	},
	DefaultPoolAddr: {
		flagname:  "defaultpooladdr",
		flagusage: "Default Pool URL",
		envname:   "DefaultPoolAddr",
		defval:    "stratum+tcp://127.0.0.1:33334/",
		configval: nil,
		envval:    nil,
		flagval:   nil,
	},
	ConfigRESTPort: {
		flagname:   "rest_port",
		flagusage:  "REST API Port",
		envname:    "RESTPORT",
		configname: "externalAPI.port",
		defval:     "8080",
		configval:  nil,
		envval:     nil,
		flagval:    nil,
	},
	DisableConnection: {
		flagname:  "disableconnection",
		flagusage: "Disable the connection manager",
		envname:   "DISABLECONNECTION",
		defval:    "false",
		configval: nil,
		envval:    nil,
		flagval:   nil,
	},
	DisableContract: {
		flagname:  "disablecontract",
		flagusage: "Disable the contract manager",
		envname:   "DISABLECONTRACT",
		defval:    "false",
		configval: nil,
		envval:    nil,
		flagval:   nil,
	},
	DisableSchedule: {
		flagname:  "disableschedule",
		flagusage: "Disable the schedule manager",
		envname:   "DISABLESCHEDULE",
		defval:    "false",
		configval: nil,
		envval:    nil,
		flagval:   nil,
	},
	DisableStratumv1: {
		flagname:  "disablestratumv1",
		flagusage: "Disable the Stratum V1 Protocol",
		envname:   "DISABLESTRATUMV1",
		defval:    "false",
		configval: nil,
		envval:    nil,
		flagval:   nil,
	},
	DisableAPI: {
		flagname:  "disableapi",
		flagusage: "Disable the external api",
		envname:   "DISABLEAPI",
		defval:    "false",
		configval: nil,
		envval:    nil,
		flagval:   nil,
	},
}
