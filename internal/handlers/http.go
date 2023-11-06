package handlers

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/config"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/contractmanager"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	hrcontract "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/contract"
	hr "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"golang.org/x/exp/slices"
)

type Proxy interface {
	SetDest(ctx context.Context, newDestURL *url.URL, onSubmit func(diff float64)) error
}

type ContractFactory func(contractData *hashrate.Terms) (resources.Contract, error)

type HTTPHandler struct {
	globalHashrate         *hr.GlobalHashrate
	allocator              *allocator.Allocator
	contractManager        *contractmanager.ContractManager
	hashrateCounterDefault string
	publicUrl              *url.URL
	pubKey                 string
	log                    interfaces.ILogger
}

func NewHTTPHandler(allocator *allocator.Allocator, contractManager *contractmanager.ContractManager, globalHashrate *hr.GlobalHashrate, publicUrl *url.URL, hashrateCounter string, log interfaces.ILogger) *gin.Engine {
	handl := &HTTPHandler{
		allocator:              allocator,
		contractManager:        contractManager,
		globalHashrate:         globalHashrate,
		publicUrl:              publicUrl,
		hashrateCounterDefault: hashrateCounter,
		log:                    log,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.GET("/healthcheck", handl.HealthCheck)
	r.GET("/miners", handl.GetMiners)
	r.GET("/contracts", handl.GetContracts)
	r.GET("/contracts/:ID/logs", handl.GetDeliveryLogs)
	r.GET("/workers", handl.GetWorkers)

	r.POST("/change-dest", handl.ChangeDest)
	r.POST("/contracts", handl.CreateContract)

	err := r.SetTrustedProxies(nil)
	if err != nil {
		panic(err)
	}

	return r
}

func (h *HTTPHandler) HealthCheck(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"status":  "healthy",
		"version": config.BuildVersion,
	})
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
		_ = ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	hrGHS, err := strconv.ParseInt(ctx.Query("hrGHS"), 10, 0)
	if err != nil {
		_ = ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	duration, err := time.ParseDuration(ctx.Query("duration"))
	if err != nil {
		_ = ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	now := time.Now()
	destEnc, err := lib.EncryptString(dest.String(), h.pubKey)
	if err != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	terms := hashrate.NewTerms(
		lib.GetRandomAddr().String(),
		lib.GetRandomAddr().String(),
		lib.GetRandomAddr().String(),
		now,
		duration,
		float64(hrGHS)*1e9,
		big.NewInt(0),
		hashrate.BlockchainStateRunning,
		false,
		big.NewInt(0),
		false,
		0,
		destEnc,
	)
	h.contractManager.AddContract(context.Background(), terms)

	ctx.JSON(200, gin.H{"status": "ok"})
}

func (c *HTTPHandler) GetContracts(ctx *gin.Context) {
	data := []Contract{}
	c.contractManager.GetContracts().Range(func(item resources.Contract) bool {
		contract := c.mapContract(item)
		// 		contract.Miners = miners
		data = append(data, *contract)
		return true
	})

	slices.SortStableFunc(data, func(a Contract, b Contract) bool {
		return a.ID < b.ID
	})

	ctx.JSON(200, data)
}

func (c *HTTPHandler) GetDeliveryLogs(ctx *gin.Context) {
	contractID := ctx.Param("ID")
	if contractID == "" {
		ctx.JSON(400, gin.H{"error": "contract id is required"})
		return
	}
	contract, ok := c.contractManager.GetContracts().Load(contractID)
	if !ok {
		ctx.JSON(404, gin.H{"error": "contract not found"})
		return
	}

	sellerContract, ok := contract.(*hrcontract.ControllerSeller)
	if !ok {
		ctx.JSON(400, gin.H{"error": "contract is not seller contract"})
		return
	}
	logs, err := sellerContract.GetDeliveryLogs()
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	err = writeHTML(ctx.Writer, logs)
	if err != nil {
		c.log.Errorf("failed to write logs: %s", err)
		_ = ctx.Error(err)
		ctx.Abort()
	}
	return
}

func writeHTML(w io.Writer, logs []hrcontract.DeliveryLogEntry) error {
	header := []string{
		"TimestampUnix",
		"ActualGHS",
		"FullMinersGHS",
		"FullMinersNumber",
		"PartialMinersGHS",
		"PartialMinersNumber",
		"SharesSubmitted",
		"UnderDeliveryGHS",
		"GlobalHashrateGHS",
		"GlobalUnderDeliveryGHS",
		"GlobalError",
		"NextCyclePartialDeliveryTargetGHS",
	}

	// header
	_, _ = w.Write([]byte(`
		<html>
			<style>
				table {
					font-family: monospace;
					border-collapse: collapse;
					font-size: 12px;
					border: 1px solid #333;
				}
				th, td {
					padding: 3px;
					border: 1px solid #333;
				}
			</style>
			<body>
				<table>`))

	// table header
	_, _ = w.Write([]byte("<tr>"))
	for _, h := range header {
		err := writeTableRow("th", w, h)
		if err != nil {
			return err
		}
	}
	_, _ = w.Write([]byte("</tr>"))

	// table body
	for _, entry := range logs {
		_, _ = w.Write([]byte("<tr>"))
		err := writeTableRow("td", w,
			formatTime(entry.Timestamp),
			fmt.Sprint(entry.ActualGHS),
			fmt.Sprint(entry.FullMinersGHS),
			fmt.Sprint(entry.FullMinersNumber),
			fmt.Sprint(entry.PartialMinersGHS),
			fmt.Sprint(entry.PartialMinersNumber),
			fmt.Sprint(entry.SharesSubmitted),
			fmt.Sprint(entry.UnderDeliveryGHS),
			fmt.Sprint(entry.GlobalHashrateGHS),
			fmt.Sprint(entry.GlobalUnderDeliveryGHS),
			fmt.Sprintf("%.2f", entry.GlobalError),
			fmt.Sprint(entry.NextCyclePartialDeliveryTargetGHS),
		)
		if err != nil {
			return err
		}
		_, _ = w.Write([]byte("</tr>"))
	}

	// footer
	_, _ = w.Write([]byte(`
				</table>
			</body>
		</html>`))

	return nil
}

func writeTableRow(tag string, w io.Writer, values ...string) error {
	for _, value := range values {
		_, err := w.Write([]byte(fmt.Sprintf("<%s>%s</%s>", tag, value, tag)))
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *HTTPHandler) GetMiners(ctx *gin.Context) {
	Miners := []Miner{}

	var (
		TotalHashrateGHS float64
		UsedHashrateGHS  float64

		TotalMiners       int
		BusyMiners        int
		PartialBusyMiners int
		FreeMiners        int
		VettingMiners     int
	)

	c.allocator.GetMiners().Range(func(m *allocator.Scheduler) bool {
		hrGHS, ok := m.GetHashrate().GetHashrateAvgGHSCustom(c.hashrateCounterDefault)
		if !ok {
			c.log.DPanicf("hashrate counter not found, %s", c.hashrateCounterDefault)
		} else {
			TotalHashrateGHS += hrGHS
		}

		hrGHS, ok = m.GetUsedHashrate().GetHashrateAvgGHSCustom(c.hashrateCounterDefault)
		if !ok {
			c.log.DPanicf("hashrate counter not found, %s", c.hashrateCounterDefault)
		} else {
			UsedHashrateGHS += hrGHS
		}

		TotalMiners += 1

		switch m.GetStatus() {
		case allocator.MinerStatusFree:
			FreeMiners += 1
		case allocator.MinerStatusVetting:
			VettingMiners += 1
		case allocator.MinerStatusBusy:
			BusyMiners += 1
		case allocator.MinerStatusPartialBusy:
			PartialBusyMiners += 1
		}

		miner := c.MapMiner(m)
		Miners = append(Miners, *miner)

		return true
	})

	slices.SortStableFunc(Miners, func(a Miner, b Miner) bool {
		return a.ID < b.ID
	})

	res := &MinersResponse{
		TotalMiners:       TotalMiners,
		VettingMiners:     VettingMiners,
		FreeMiners:        FreeMiners,
		PartialBusyMiners: PartialBusyMiners,
		BusyMiners:        BusyMiners,

		TotalHashrateGHS:     int(TotalHashrateGHS),
		AvailableHashrateGHS: int(TotalHashrateGHS - UsedHashrateGHS),
		UsedHashrateGHS:      int(UsedHashrateGHS),

		Miners: Miners,
	}

	ctx.JSON(200, res)
}

func (c *HTTPHandler) MapMiner(m *allocator.Scheduler) *Miner {
	return &Miner{
		Resource: Resource{
			Self: c.publicUrl.JoinPath(fmt.Sprintf("/miners/%s", m.ID())).String(),
		},
		ID:                    m.ID(),
		WorkerName:            m.GetWorkerName(),
		Status:                m.GetStatus().String(),
		CurrentDifficulty:     int(m.GetCurrentDifficulty()),
		HashrateAvgGHS:        mapHRToInt(m),
		CurrentDestination:    m.GetCurrentDest().String(),
		ConnectedAt:           m.GetConnectedAt().Format(time.RFC3339),
		Stats:                 m.GetStats(),
		Uptime:                formatDuration(m.GetUptime()),
		ActivePoolConnections: m.GetDestConns(),
		Destinations:          m.GetDestinations(),
	}
}

func (c *HTTPHandler) GetWorkers(ctx *gin.Context) {
	workers := []*Worker{}

	c.globalHashrate.Range(func(item *hr.WorkerHashrateModel) bool {
		worker := &Worker{
			WorkerName: item.ID(),
			Hashrate:   item.GetHashrateAvgGHSAll(),
		}
		workers = append(workers, worker)
		return true
	})

	slices.SortStableFunc(workers, func(a *Worker, b *Worker) bool {
		return a.WorkerName < b.WorkerName
	})

	ctx.JSON(200, workers)
}

func (p *HTTPHandler) mapContract(item resources.Contract) *Contract {
	return &Contract{
		Resource: Resource{
			Self: p.publicUrl.JoinPath(fmt.Sprintf("/contracts/%s", item.ID())).String(),
		},
		Role:                    item.Role().String(),
		Stage:                   item.ValidationStage().String(),
		ID:                      item.ID(),
		BuyerAddr:               item.Buyer(),
		SellerAddr:              item.Seller(),
		ResourceEstimatesTarget: roundResourceEstimates(item.ResourceEstimates()),
		ResourceEstimatesActual: roundResourceEstimates(item.ResourceEstimatesActual()),
		PriceLMR:                LMRWithDecimalsToLMR(item.Price()),
		Duration:                formatDuration(item.Duration()),

		IsDeleted:      item.IsDeleted(),
		BalanceLMR:     LMRWithDecimalsToLMR(item.Balance()),
		HasFutureTerms: item.HasFutureTerms(),
		Version:        item.Version(),

		StartTimestamp:    formatTime(item.StartTime()),
		EndTimestamp:      formatTime(item.EndTime()),
		Elapsed:           formatDuration(item.Elapsed()),
		ApplicationStatus: item.State().String(),
		BlockchainStatus:  item.BlockchainState().String(),
		Dest:              item.Dest(),
		Miners:            p.allocator.GetMinersFulfillingContract(item.ID()),
	}
}

// TimePtrToStringPtr converts nullable time to nullable string
func TimePtrToStringPtr(t *time.Time) *string {
	if t != nil {
		a := formatTime(*t)
		return &a
	}
	return nil
}

func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

func DurationPtrToStringPtr(t *time.Duration) *string {
	if t != nil {
		a := formatDuration(*t)
		return &a
	}
	return nil
}

func mapHRToInt(m *allocator.Scheduler) map[string]int {
	hrFloat := m.GetHashrate().GetHashrateAvgGHSAll()
	hrInt := make(map[string]int, len(hrFloat))
	for k, v := range hrFloat {
		hrInt[k] = int(v)
	}
	return hrInt
}

func formatDuration(dur time.Duration) string {
	return dur.Round(time.Second).String()
}

func roundResourceEstimates(estimates map[string]float64) map[string]int {
	res := make(map[string]int, len(estimates))
	for k, v := range estimates {
		res[k] = int(v)
	}
	return res
}

// LMRWithDecimalsToLMR converts LMR with decimals to LMR without decimals
func LMRWithDecimalsToLMR(LMRWithDecimals *big.Int) float64 {
	v, _ := lib.NewRat(LMRWithDecimals, big.NewInt(1e8)).Float64()
	return v
}
