package api

import (
	"context"
	"math"
	"math/big"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/miner"
	"golang.org/x/exp/slices"
)

type ApiController struct {
	miners    interfaces.ICollection[miner.MinerScheduler]
	contracts interfaces.ICollection[contractmanager.IContractModel]
}

type Miner struct {
	ID                 string
	Status             string
	TotalHashrateGHS   int
	HashrateAvgGHS     HashrateAvgGHS
	Destinations       []DestItem
	CurrentDestination string
	CurrentDifficulty  int
	WorkerName         string
	ConnectedAt        string
	UptimeSeconds      int
}

type HashrateAvgGHS struct {
	T5m  int `json:"5m"`
	T30m int `json:"30m"`
	T1h  int `json:"1h"`
}

type DestItem struct {
	URI      string
	Fraction float64
}

type Contract struct {
	ID              string
	BuyerAddr       string
	SellerAddr      string
	HashrateGHS     int
	DurationSeconds int
	StartTimestamp  *string
	EndTimestamp    *string
	State           string
	Dest            string
	// Miners         []string
}

func NewApiController(miners interfaces.ICollection[miner.MinerScheduler], contracts interfaces.ICollection[contractmanager.IContractModel], log interfaces.ILogger, gs *contractmanager.GlobalSchedulerService) *gin.Engine {
	r := gin.Default()
	controller := ApiController{
		miners:    miners,
		contracts: contracts,
	}

	r.GET("/miners", func(ctx *gin.Context) {
		data := controller.GetMiners()
		ctx.JSON(http.StatusOK, data)
	})

	r.GET("/contracts", func(ctx *gin.Context) {
		data := controller.GetContracts()
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
			Addr:                   common.BigToAddress(big.NewInt(rand.Int63())),
			State:                  blockchain.ContractBlockchainStateRunning,
			Price:                  0,
			Speed:                  hrGHS * int64(math.Pow10(9)),
			Length:                 int64(duration.Seconds()),
			Dest:                   dest,
			StartingBlockTimestamp: time.Now().Unix(),
		}, nil, gs, log, hashrate.NewHashrate(log))

		go func() {
			err := contract.FulfillContract(context.Background())
			if err != nil {
				log.Errorf("error during fulfillment of the test contract: %s", err)
			}
		}()

		contracts.Store(contract)
	})

	return r
}

func (c *ApiController) GetMiners() []Miner {
	data := []Miner{}
	c.miners.Range(func(miner miner.MinerScheduler) bool {
		destItems := []DestItem{}
		dest := miner.GetDestSplit()
		for _, item := range dest.Iter() {
			destItems = append(destItems, DestItem{
				URI:      item.Dest.String(),
				Fraction: item.Percentage,
			})
		}

		hashrate := miner.GetHashRate()
		data = append(data, Miner{
			ID:                miner.GetID(),
			Status:            miner.GetStatus().String(),
			TotalHashrateGHS:  miner.GetHashRateGHS(),
			CurrentDifficulty: miner.GetCurrentDifficulty(),
			Destinations:      destItems,
			HashrateAvgGHS: HashrateAvgGHS{
				T5m:  hashrate.GetHashrate5minAvgGHS(),
				T30m: hashrate.GetHashrate30minAvgGHS(),
				T1h:  hashrate.GetHashrate1hAvgGHS(),
			},
			CurrentDestination: miner.GetCurrentDest().String(),
			WorkerName:         miner.GetWorkerName(),
			ConnectedAt:        miner.GetConnectedAt().Format(time.RFC3339),
			UptimeSeconds:      int(miner.GetUptime().Seconds()),
		})
		return true
	})

	slices.SortStableFunc(data, func(a Miner, b Miner) bool {
		return a.ID < b.ID
	})

	return data
}

func (c *ApiController) changeDestAll(destStr string) error {
	dest, err := lib.ParseDest(destStr)
	if err != nil {
		return err
	}

	c.miners.Range(func(miner miner.MinerScheduler) bool {
		err = miner.ChangeDest(dest)
		return err == nil
	})

	return err
}

func (c *ApiController) GetContracts() []Contract {
	data := []Contract{}
	c.contracts.Range(func(item contractmanager.IContractModel) bool {
		data = append(data, Contract{
			ID:              item.GetID(),
			BuyerAddr:       item.GetBuyerAddress(),
			SellerAddr:      item.GetSellerAddress(),
			HashrateGHS:     item.GetHashrateGHS(),
			DurationSeconds: int(item.GetDuration().Seconds()),
			StartTimestamp:  TimePtrToStringPtr(item.GetStartTime()),
			EndTimestamp:    TimePtrToStringPtr(item.GetEndTime()),
			State:           MapContractState(item.GetState()),
			Dest:            item.GetDest().String(),
		})
		return true
	})

	slices.SortStableFunc(data, func(a Contract, b Contract) bool {
		return a.ID < b.ID
	})
	return data
}

func MapContractState(state contractmanager.ContractState) string {
	switch state {
	case contractmanager.ContractStateAvailable:
		return "available"
	case contractmanager.ContractStatePurchased:
		return "purchased"
	case contractmanager.ContractStateRunning:
		return "running"
	case contractmanager.ContractStateClosed:
		return "closed"
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
