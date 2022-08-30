package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/miner"
)

type ApiController struct {
	miners    interfaces.ICollection[miner.MinerScheduler]
	contracts interfaces.ICollection[*contractmanager.Contract]
}

type Miner struct {
	ID                 string
	TotalHashrateGHS   int
	Destinations       []DestItem
	CurrentDestination string
}

type DestItem struct {
	URI        string
	Percentage int
}

type Contract struct {
	ID             string
	BuyerAddr      string
	SellerAddr     string
	HashrateGHS    int
	StartTimestamp string
	EndTimestamp   string
	State          string
	// Miners         []string
}

func NewApiController(miners interfaces.ICollection[miner.MinerScheduler], contracts interfaces.ICollection[*contractmanager.Contract]) *gin.Engine {
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
			Destinations:       destItems,
			CurrentDestination: miner.GetCurrentDest().String(),
		})
		return true
	})

	return data
}

func (c *ApiController) GetContracts() []Contract {
	data := []Contract{}
	c.contracts.Range(func(item *contractmanager.Contract) bool {

		data = append(data, Contract{
			ID:             item.GetID(),
			BuyerAddr:      item.GetBuyerAddress(),
			SellerAddr:     item.GetSellerAddress(),
			HashrateGHS:    item.GetHashrateGHS(),
			StartTimestamp: item.GetStartTime().Format(time.RFC3339),
			EndTimestamp:   item.GetEndTime().Format(time.RFC3339),
			State:          MapContractState(item.GetState()),
		})
		return true
	})

	return data
}

func MapContractState(state contractmanager.ContractState) string {
	switch state {
	case contractmanager.ContractStateCreated:
		return "created"
	case contractmanager.ContractStatePurchased:
		return "purchased"
	case contractmanager.ContractStateRunning:
		return "running"
	case contractmanager.ContractStateClosed:
		return "closed"
	}
	return "unknown"
}
