package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/data"
	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/miner"
	"gitlab.com/TitanInd/hashrouter/protocol"
	"golang.org/x/exp/slices"
)

type Resource struct {
	Self string
}

type ApiController struct {
	miners             interfaces.ICollection[miner.MinerScheduler]
	contracts          interfaces.ICollection[contractmanager.IContractModel]
	defaultDestination interfaces.IDestination

	publicUrl *url.URL
}

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
	Status                string
	TotalHashrateGHS      int
	HashrateAvgGHS        HashrateAvgGHS
	Destinations          *[]DestItem
	UpcomingDestinations  *[]DestItem
	CurrentDestination    string
	CurrentDifficulty     int
	WorkerName            string
	ConnectedAt           string
	UptimeSeconds         int
	ActivePoolConnections *map[string]string `json:",omitempty"`
	History               *[]HistoryItem     `json:",omitempty"`
	IsFaulty              bool
}

type HashrateAvgGHS struct {
	T5m   int `json:"5m"`
	T30m  int `json:"30m"`
	T1h   int `json:"1h"`
	SMA9m int `json:"SMA9m"`
}

type DestItem struct {
	ContractID  string
	URI         string
	Fraction    float64
	HashrateGHS int
}

type Contract struct {
	Resource

	ID                   string
	BuyerAddr            string
	SellerAddr           string
	HashrateGHS          int
	DeliveredHashrateGHS *HashrateAvgGHS
	DurationSeconds      int
	StartTimestamp       *string
	EndTimestamp         *string
	ApplicationStatus    string
	BlockchainStatus     string
	Dest                 string
	History              *[]HistoryItem `json:",omitempty"`
	Miners               []Miner
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

func NewApiController(miners interfaces.ICollection[miner.MinerScheduler], contracts interfaces.ICollection[contractmanager.IContractModel], log interfaces.ILogger, gs *contractmanager.GlobalSchedulerV2, isBuyer bool, hashrateDiffThreshold float64, validationBufferPeriod time.Duration, defaultDestination interfaces.IDestination, apiPublicUrl string, contractCycleDuration time.Duration) *gin.Engine {
	publicUrl, _ := url.Parse(apiPublicUrl)

	controller := ApiController{
		miners:             miners,
		contracts:          contracts,
		defaultDestination: defaultDestination,
		publicUrl:          publicUrl,
	}

	r := gin.Default()

	r.GET("/healthcheck", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

	r.GET("/miners", func(ctx *gin.Context) {
		data := controller.GetMiners()
		ctx.JSON(http.StatusOK, data)
	})

	r.GET("/miners/:id", func(ctx *gin.Context) {
		data, ok := controller.GetMiner(ctx.Param("id"))
		if !ok {
			ctx.Status(http.StatusNotFound)
			return
		}
		ctx.JSON(http.StatusOK, data)
	})

	r.GET("/contracts", func(ctx *gin.Context) {
		data := controller.GetContracts()
		ctx.JSON(http.StatusOK, data)
	})

	r.GET("/contracts/:id", func(ctx *gin.Context) {
		data, ok := controller.GetContract(ctx.Param("id"))
		if !ok {
			ctx.Status(http.StatusNotFound)
			return
		}
		ctx.JSON(http.StatusOK, data)
	})

	// for tests
	r.POST("/miners/change-dest", func(ctx *gin.Context) {
		dest := ctx.Query("dest")
		if dest == "" {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		err := controller.changeDestAll(dest)

		if err != nil {
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		ctx.Status(http.StatusOK)
	})

	// for tests
	r.POST("/contracts", func(ctx *gin.Context) {
		dest, err := lib.ParseDest(ctx.Query("dest"))
		if err != nil {
			ctx.AbortWithStatus(http.StatusBadRequest)
		}
		hrGHS, err := strconv.ParseInt(ctx.Query("hrGHS"), 10, 0)
		if err != nil {
			ctx.AbortWithStatus(http.StatusBadRequest)
		}
		duration, err := time.ParseDuration(ctx.Query("duration"))
		if err != nil {
			ctx.AbortWithStatus(http.StatusBadRequest)
		}
		contract := contractmanager.NewContract(blockchain.ContractData{
			Addr:                   lib.GetRandomAddr(),
			State:                  blockchain.ContractBlockchainStateRunning,
			Price:                  0,
			Speed:                  hrGHS * int64(math.Pow10(9)),
			Length:                 int64(duration.Seconds()),
			Dest:                   dest,
			StartingBlockTimestamp: time.Now().Unix(),
		}, nil, gs, log, hashrate.NewHashrateV2(hashrate.NewSma(9*time.Minute)), hashrateDiffThreshold, validationBufferPeriod, controller.defaultDestination, contractCycleDuration)

		go func() {
			err := contract.FulfillContract(context.Background())
			if err != nil {
				log.Errorf("error during fulfillment of the test contract: %s", err)
				contract.Stop(context.Background())
			}
		}()

		contracts.Store(contract)
	})

	// for tests
	r.POST("/contracts/:id/dest", func(ctx *gin.Context) {
		dest, err := lib.ParseDest(ctx.Query("dest"))
		if err != nil {
			ctx.AbortWithStatus(http.StatusBadRequest)
		}
		contract, ok := controller.contracts.Load(ctx.Param("id"))
		if !ok {
			ctx.Status(http.StatusNotFound)
			return
		}
		if contract.IsBuyer() {
			ctx.Status(http.StatusConflict)
			return
		}
		contract.(*contractmanager.BTCHashrateContractSeller).SetDest(dest)
	})

	return r
}

func (c *ApiController) GetMiners() *MinersResponse {
	Miners := []Miner{}

	var (
		TotalHashrateGHS int
		UsedHashrateGHS  int

		TotalMiners   int
		BusyMiners    int
		FreeMiners    int
		VettingMiners int
		FaultyMiners  int
	)

	c.miners.Range(func(m miner.MinerScheduler) bool {
		_, usedHR := mapDestItems(m.GetCurrentDestSplit(), m.GetHashRateGHS())
		UsedHashrateGHS += usedHR

		hashrate := m.GetHashRate()
		TotalHashrateGHS += hashrate.GetHashrate5minAvgGHS()
		TotalMiners += 1

		switch m.GetStatus() {
		case miner.MinerStatusFree:
			FreeMiners += 1
		case miner.MinerStatusVetting:
			VettingMiners += 1
		case miner.MinerStatusBusy:
			BusyMiners += 1
		}

		if m.IsFaulty() {
			FaultyMiners += 1
		}

		miner := c.MapMiner(m)
		miner.ActivePoolConnections = c.mapPoolConnection(m)
		Miners = append(Miners, *miner)

		return true
	})

	slices.SortStableFunc(Miners, func(a Miner, b Miner) bool {
		return a.ID < b.ID
	})

	return &MinersResponse{
		TotalMiners:   TotalMiners,
		BusyMiners:    BusyMiners,
		FreeMiners:    FreeMiners,
		VettingMiners: VettingMiners,

		TotalHashrateGHS:     TotalHashrateGHS,
		AvailableHashrateGHS: TotalHashrateGHS - UsedHashrateGHS,
		UsedHashrateGHS:      UsedHashrateGHS,

		Miners: Miners,
	}
}

func (*ApiController) mapPoolConnection(m miner.MinerScheduler) *map[string]string {
	ActivePoolConnections := make(map[string]string)

	m.RangeDestConn(func(key, value any) bool {
		k := value.(*protocol.StratumV1PoolConn)
		ActivePoolConnections[key.(string)] = k.GetDeadline().Format(time.RFC3339)
		return true
	})

	return &ActivePoolConnections
}

func mapDestItems(dest *miner.DestSplit, hrGHS int) (*[]DestItem, int) {
	destItems := []DestItem{}
	UsedHashrateGHS := 0

	if dest == nil {
		return nil, 0
	}

	for _, item := range dest.Iter() {
		HashrateGHS := int(item.Fraction * float64(hrGHS))

		destItems = append(destItems, DestItem{
			ContractID:  item.ID,
			URI:         item.Dest.String(),
			Fraction:    item.Fraction,
			HashrateGHS: HashrateGHS,
		})

		UsedHashrateGHS += HashrateGHS
	}
	return &destItems, UsedHashrateGHS
}

func (c *ApiController) GetMiner(ID string) (*Miner, bool) {
	m, ok := c.miners.Load(ID)
	if !ok {
		return nil, false
	}

	var history []HistoryItem

	m.RangeHistory(func(item miner.HistoryItem) bool {
		history = append(history, HistoryItem{
			MinerID:         m.GetID(),
			ContractID:      item.ContractID,
			Dest:            item.Dest.String(),
			DurationMs:      item.Duration.Milliseconds(),
			DurationString:  item.Duration.String(),
			TimestampUnixMs: item.Timestamp.UnixMilli(),
			TimestampString: item.Timestamp.Format(time.RFC3339Nano),
		})
		return true
	})

	miner := c.MapMiner(m)

	miner.UpcomingDestinations, _ = mapDestItems(m.GetUpcomingDestSplit(), m.GetHashRateGHS())
	miner.ActivePoolConnections = c.mapPoolConnection(m)
	miner.History = &history

	return miner, true
}

func (c *ApiController) changeDestAll(destStr string) error {
	dest, err := lib.ParseDest(destStr)
	if err != nil {
		return err
	}

	c.miners.Range(func(miner miner.MinerScheduler) bool {
		err = miner.ChangeDest(context.TODO(), dest, fmt.Sprintf("api-change-dest-all-%s", lib.GetRandomAddr()), nil)
		return err == nil
	})

	return err
}

func (c *ApiController) GetContracts() []Contract {
	snap := CreateCurrentMinerSnapshot(c.miners)

	data := []Contract{}
	c.contracts.Range(func(item contractmanager.IContractModel) bool {
		minerIDs := []string{}
		m, ok := snap.Contract(item.GetID())
		if ok {
			minerIDs = m.IDs()
		}

		var miners []Miner

		for _, k := range minerIDs {
			m, ok := c.miners.Load(k)
			if !ok {
				continue
			}

			miners = append(miners, *c.MapMiner(m))
		}

		contract := c.MapContract(item)
		contract.Miners = miners

		data = append(data, *contract)
		return true
	})

	slices.SortStableFunc(data, func(a Contract, b Contract) bool {
		return a.ID < b.ID
	})
	return data
}

func (c *ApiController) GetContract(ID string) (*Contract, bool) {
	contract, ok := c.contracts.Load(ID)
	if !ok {
		return nil, false
	}

	var history []HistoryItem
	var miners []Miner

	c.miners.Range(func(mn miner.MinerScheduler) bool {
		miners = append(miners, *c.MapMiner(mn))

		mn.RangeHistoryContractID(ID, func(item miner.HistoryItem) bool {
			history = append(history, HistoryItem{
				MinerID:         mn.GetID(),
				ContractID:      item.ContractID,
				Dest:            item.Dest.String(),
				DurationMs:      item.Duration.Milliseconds(),
				DurationString:  item.Duration.String(),
				TimestampUnixMs: item.Timestamp.UnixMilli(),
				TimestampString: item.Timestamp.Format(time.RFC3339Nano),
			})
			return true
		})
		return true
	})

	slices.SortFunc(history, func(a, b HistoryItem) bool {
		return a.TimestampUnixMs < b.TimestampUnixMs
	})

	item := c.MapContract(contract)
	item.History = &history

	return item, true
}

func (c *ApiController) MapMiner(m miner.MinerScheduler) *Miner {
	hashrate := m.GetHashRate()
	destItems, _ := mapDestItems(m.GetCurrentDestSplit(), m.GetHashRateGHS())
	upcomingDest, _ := mapDestItems(m.GetUpcomingDestSplit(), m.GetHashRateGHS())
	SMA9m, _ := hashrate.GetHashrateAvgGHSCustom(0)
	return &Miner{
		Resource: Resource{
			Self: c.publicUrl.JoinPath(fmt.Sprintf("/miners/%s", m.GetID())).String(),
		},
		ID:                m.GetID(),
		Status:            m.GetStatus().String(),
		TotalHashrateGHS:  m.GetHashRateGHS(),
		CurrentDifficulty: m.GetCurrentDifficulty(),
		HashrateAvgGHS: HashrateAvgGHS{
			T5m:   hashrate.GetHashrate5minAvgGHS(),
			T30m:  hashrate.GetHashrate30minAvgGHS(),
			T1h:   hashrate.GetHashrate1hAvgGHS(),
			SMA9m: SMA9m,
		},
		Destinations:         destItems,
		UpcomingDestinations: upcomingDest,
		CurrentDestination:   m.GetCurrentDest().String(),
		WorkerName:           m.GetWorkerName(),
		ConnectedAt:          m.GetConnectedAt().Format(time.RFC3339),
		UptimeSeconds:        int(m.GetUptime().Seconds()),
		IsFaulty:             m.IsFaulty(),
	}
}

func (c *ApiController) MapContract(item contractmanager.IContractModel) *Contract {
	var hashrateAvgGHS *HashrateAvgGHS
	hr := item.GetDeliveredHashrate()
	if hr != nil {
		hashrateAvgGHS = &HashrateAvgGHS{
			T5m:  hr.GetHashrate5minAvgGHS(),
			T30m: hr.GetHashrate30minAvgGHS(),
			T1h:  hr.GetHashrate1hAvgGHS(),
		}
	}
	return &Contract{
		Resource: Resource{
			Self: c.publicUrl.JoinPath(fmt.Sprintf("/contracts/%s", item.GetID())).String(),
		},
		ID:                   item.GetID(),
		BuyerAddr:            item.GetBuyerAddress(),
		SellerAddr:           item.GetSellerAddress(),
		HashrateGHS:          item.GetHashrateGHS(),
		DeliveredHashrateGHS: hashrateAvgGHS,
		DurationSeconds:      int(item.GetDuration().Seconds()),
		StartTimestamp:       TimePtrToStringPtr(item.GetStartTime()),
		EndTimestamp:         TimePtrToStringPtr(item.GetEndTime()),
		ApplicationStatus:    MapContractState(item.GetState()),
		BlockchainStatus:     item.GetStatusInternal(),
		Dest:                 item.GetDest().String(),
	}
}

func MapContractState(state contractmanager.ContractState) string {
	switch state {
	case contractmanager.ContractStateAvailable:
		return "available"
	case contractmanager.ContractStatePurchased:
		return "purchased"
	case contractmanager.ContractStateRunning:
		return "running"
	}
	return "unknown"
}

func TimePtrToStringPtr(t *time.Time) *string {
	if t != nil {
		a := t.Format(time.RFC3339)
		return &a
	}
	return nil
}

// CreateCurrentMinerSnapshot returns current state of the miners
func CreateCurrentMinerSnapshot(minerCollection interfaces.ICollection[miner.MinerScheduler]) data.AllocSnap {
	snapshot := data.NewAllocSnap()

	minerCollection.Range(func(miner miner.MinerScheduler) bool {
		if miner.IsVetting() {
			return true
		}
		if miner.IsFaulty() {
			return true
		}

		hashrateGHS := miner.GetHashRateGHS()
		minerID := miner.GetID()

		snapshot.SetMiner(minerID, hashrateGHS)

		for _, splitItem := range miner.GetCurrentDestSplit().Iter() {
			snapshot.Set(minerID, splitItem.ID, splitItem.Fraction, hashrateGHS)
		}

		return true
	})

	return snapshot
}
