package httphandlers

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
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate"
	hrcontract "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/contract"
	"golang.org/x/exp/slices"
)

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

func (c *HTTPHandler) GetContract(ctx *gin.Context) {
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

	contractData := c.mapContract(contract)
	ctx.JSON(200, contractData)
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

func (p *HTTPHandler) mapContract(item resources.Contract) *Contract {
	return &Contract{
		Resource: Resource{
			Self: p.publicUrl.JoinPath(fmt.Sprintf("/contracts/%s", item.ID())).String(),
		},
		Logs:                    p.publicUrl.JoinPath(fmt.Sprintf("/contracts/%s/logs", item.ID())).String(),
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
