package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	"golang.org/x/exp/slices"
)

type Proxy interface {
	SetDest(ctx context.Context, newDestURL *url.URL, onSubmit func(diff float64)) error
}

type HTTPHandler struct {
	allocator *allocator.Allocator
	publicUrl *url.URL
	log       interfaces.ILogger
}

func NewHTTPHandler(allocator *allocator.Allocator, publicUrl *url.URL, log interfaces.ILogger) *HTTPHandler {
	return &HTTPHandler{
		publicUrl: publicUrl,
		allocator: allocator,
		log:       log,
	}
}

func (h *HTTPHandler) ChangeDest(ctx *gin.Context) {
	urlString := ctx.Query("dest")
	if urlString == "" {
		ctx.JSON(400, gin.H{"error": "empty destination"})
		return
	}
	dest, err := url.Parse(urlString)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	miners := h.allocator.GetMiners()
	miners.Range(func(m *allocator.Scheduler) bool {
		m.SetPrimaryDest(dest)
		return true
	})

	ctx.JSON(200, gin.H{"status": "ok"})
}

func (h *HTTPHandler) CreateContract(ctx *gin.Context) {
	dest, err := url.Parse(ctx.Query("dest"))
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

	watcher := allocator.NewContractWatcher(lib.GetRandomAddr().String(), float64(hrGHS), dest, time.Now(), duration, h.allocator)

	go func() {
		err := watcher.Run(context.Background())
		if err != nil {
			h.log.Errorf("error during fulfillment of the test contract: %s", err)
		}
	}()
}

func (c *HTTPHandler) GetMiners(ctx *gin.Context) {
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

	c.allocator.GetMiners().Range(func(m *allocator.Scheduler) bool {
		// _, usedHR := mapDestItems(m.GetCurrentDestSplit(), m.GetHashRateGHS())
		// UsedHashrateGHS += usedHR

		// hashrate := m.GetHashRate()
		// TotalHashrateGHS += hashrate.GetHashrate5minAvgGHS()
		// TotalMiners += 1

		// switch m.GetStatus() {
		// case miner.MinerStatusFree:
		// 	FreeMiners += 1
		// case miner.MinerStatusVetting:
		// 	VettingMiners += 1
		// case miner.MinerStatusBusy:
		// 	BusyMiners += 1
		// }

		// if m.IsFaulty() {
		// 	FaultyMiners += 1
		// }

		miner := c.MapMiner(m)
		// miner.ActivePoolConnections = mapPoolConnection(m)
		Miners = append(Miners, *miner)

		return true
	})

	slices.SortStableFunc(Miners, func(a Miner, b Miner) bool {
		return a.ID < b.ID
	})

	res := &MinersResponse{
		TotalMiners:   TotalMiners,
		BusyMiners:    BusyMiners,
		FreeMiners:    FreeMiners,
		VettingMiners: VettingMiners,
		FaultyMiners:  FaultyMiners,

		TotalHashrateGHS:     TotalHashrateGHS,
		AvailableHashrateGHS: TotalHashrateGHS - UsedHashrateGHS,
		UsedHashrateGHS:      UsedHashrateGHS,

		Miners: Miners,
	}

	ctx.JSON(200, res)
}

func (c *HTTPHandler) MapMiner(m *allocator.Scheduler) *Miner {
	hashrate := m.HashrateGHS()
	// destItems, _ := mapDestItems(m.GetCurrentDestSplit(), m.GetHashRateGHS())
	// upcomingDest, _ := mapDestItems(m.GetUpcomingDestSplit(), m.GetHashRateGHS())
	// SMA9m, _ := hashrate.GetHashrateAvgGHSCustom(0)
	return &Miner{
		Resource: Resource{
			Self: c.publicUrl.JoinPath(fmt.Sprintf("/miners/%s", m.GetID())).String(),
		},
		ID:     m.GetID(),
		Status: m.GetStatus().String(),
		// TotalHashrateGHS:  m.GetHashRateGHS(),
		CurrentDifficulty: int(m.GetCurrentDifficulty()),
		HashrateAvgGHS: HashrateAvgGHS{
			T5m: int(hashrate),
			// T5m:   hashrate.GetHashrate5minAvgGHS(),
			// T30m:  hashrate.GetHashrate30minAvgGHS(),
			// T1h:   hashrate.GetHashrate1hAvgGHS(),
			// SMA9m: SMA9m,
		},
		// Destinations:         destItems,
		// UpcomingDestinations: upcomingDest,
		CurrentDestination: m.GetCurrentDest().String(),
		WorkerName:         m.GetWorkerName(),
		ConnectedAt:        m.GetConnectedAt().Format(time.RFC3339),
		// UptimeSeconds:      int(m.GetUptime().Seconds()),
		// IsFaulty:             m.IsFaulty(),
	}
}

// func mapDestItems(dest *miner.DestSplit, hrGHS int) (*[]DestItem, int) {
// 	destItems := []DestItem{}
// 	UsedHashrateGHS := 0

// 	if dest == nil {
// 		return nil, 0
// 	}

// 	for _, item := range dest.Iter() {
// 		HashrateGHS := int(item.Fraction * float64(hrGHS))

// 		destItems = append(destItems, DestItem{
// 			ContractID:  item.ID,
// 			URI:         item.Dest.String(),
// 			Fraction:    item.Fraction,
// 			HashrateGHS: HashrateGHS,
// 		})

// 		UsedHashrateGHS += HashrateGHS
// 	}
// 	return &destItems, UsedHashrateGHS
// }

// func mapPoolConnection(m miner.MinerScheduler) *map[string]string {
// 	ActivePoolConnections := make(map[string]string)

// 	m.RangeDestConn(func(key, value any) bool {
// 		k := value.(*protocol.StratumV1PoolConn)
// 		ActivePoolConnections[key.(string)] = k.GetCloseTimeout().Format(time.RFC3339)
// 		return true
// 	})

// 	return &ActivePoolConnections
// }
