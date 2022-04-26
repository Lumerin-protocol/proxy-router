package msgbus

type ContractState string
type MinerState string

const (
	ContAvailableState ContractState = "AvailableState"
	ContRunningState   ContractState = "RunningState"
)

// Need to figure out the IDString for this, for now it is just a string
type IDString string
type ConfigID IDString
type ContractManagerConfigID IDString
type NodeOperatorID IDString
type ContractID IDString

// Do we still need this with the config package in place?

type ConfigInfo struct {
	ID           ConfigID
	DefaultDest  DestID
	NodeOperator NodeOperatorID
}

type ContractManagerConfig struct {
	ID                  ContractManagerConfigID
	Mnemonic            string
	AccountIndex        int
	EthNodeAddr         string
	ClaimFunds          bool
	TimeThreshold       int
	CloneFactoryAddress string
	LumerinTokenAddress string
	ValidatorAddress    string
	ProxyAddress        string
}

type NodeOperator struct {
	ID                     NodeOperatorID
	IsBuyer                bool
	DefaultDest            DestID
	EthereumAccount        string
	TotalAvailableHashRate int
	UnusedHashRate         int
	Contracts              map[ContractID]ContractState
}

type Contract struct {
	IsSeller               bool
	ID                     ContractID
	State                  ContractState
	Buyer                  string
	Price                  int
	Limit                  int
	Speed                  int
	Length                 int
	StartingBlockTimestamp int
	Dest                   DestID
}
