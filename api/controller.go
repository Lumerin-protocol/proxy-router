package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
	"gitlab.com/TitanInd/hashrouter/miner"
)

type ApiController struct {
	miners    interfaces.ICollection[miner.MinerScheduler]
	contracts interfaces.ICollection[contractmanager.IContractModel]
}

type Miner struct {
	ID                 string
	TotalHashrateGHS   int
	Destinations       []DestItem
	CurrentDestination string
	CurrentDifficulty  int
	WorkerName         string
}

type DestItem struct {
	URI        string
	Percentage int
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

func NewApiController(miners interfaces.ICollection[miner.MinerScheduler], contracts interfaces.ICollection[contractmanager.IContractModel]) *gin.Engine {
	r := gin.Default()
	controller := ApiController{
		miners:    miners,
		contracts: contracts,
	}

	r.GET("/miners", func(ctx *gin.Context) {
		data := controller.GetMiners()
		ctx.JSON(http.StatusOK, data)
	})

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

	r.GET("/contracts", func(ctx *gin.Context) {
		data := controller.GetContracts()
		ctx.JSON(http.StatusOK, data)
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
				URI:        item.Dest.String(),
				Percentage: int(item.Percentage),
			})
		}
		data = append(data, Miner{
			ID:                 miner.GetID(),
			TotalHashrateGHS:   miner.GetHashRateGHS(),
			CurrentDifficulty:  miner.GetCurrentDifficulty(),
			Destinations:       destItems,
			CurrentDestination: miner.GetCurrentDest().String(),
			WorkerName:         miner.GetWorkerName(),
		})
		return true
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
		var StartTimestamp *string
		var EndTimestamp *string

		if item.GetStartTime() != nil {
			*StartTimestamp = item.GetStartTime().Format(time.RFC3339)
		}

		if item.GetEndTime() != nil {
			*EndTimestamp = item.GetEndTime().Format(time.RFC3339)
		}

		data = append(data, Contract{
			ID:              item.GetID(),
			BuyerAddr:       item.GetBuyerAddress(),
			SellerAddr:      item.GetSellerAddress(),
			HashrateGHS:     item.GetHashrateGHS(),
			DurationSeconds: int(item.GetDuration().Seconds()),
			StartTimestamp:  StartTimestamp,
			EndTimestamp:    EndTimestamp,
			State:           MapContractState(item.GetState()),
			Dest:            item.GetDest().String(),
		})
		return true
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
