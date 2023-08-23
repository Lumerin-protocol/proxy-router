package handlers

type MinersResponse struct {
	TotalHashrateGHS     int
	UsedHashrateGHS      int
	AvailableHashrateGHS int

	TotalMiners   int
	BusyMiners    int
	FreeMiners    int
	VettingMiners int
	FaultyMiners  int

	Miners []Miner
}

type Miner struct {
	Resource

	ID                    string
	WorkerName            string
	Status                string
	HashrateAvgGHS        map[string]int
	Destinations          *[]DestItem
	CurrentDestination    string
	CurrentDifficulty     int
	ConnectedAt           string
	UptimeSeconds         int
	ActivePoolConnections *map[string]string `json:",omitempty"`
	History               *[]HistoryItem     `json:",omitempty"`
	IsFaulty              bool
	Stats                 interface{}
}

type Contract struct {
	Resource

	ID                      string
	BuyerAddr               string
	SellerAddr              string
	ResourceEstimatesTarget map[string]float64
	ResourceEstimatesActual map[string]float64

	DurationSeconds   int
	StartTimestamp    *string
	EndTimestamp      *string
	ApplicationStatus string
	BlockchainStatus  string
	Dest              string
	History           *[]HistoryItem `json:",omitempty"`
	Miners            []Miner
}

type Resource struct {
	Self string
}

type HistoryItem struct {
	MinerID         string
	ContractID      string
	Dest            string
	DurationMs      int64
	DurationString  string
	TimestampUnixMs int64
	TimestampString string
}

type DestItem struct {
	ContractID  string
	URI         string
	Fraction    float64
	HashrateGHS int
}
