package httphandlers

import (
	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate/hashrate"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

func (c *HTTPHandler) GetWorkers(ctx *gin.Context) {
	Workers := []*Worker{}

	c.globalHashrate.Range(func(w *hashrate.WorkerHashrateModel) bool {
		Workers = append(Workers, &Worker{
			WorkerName: w.ID(),
			Hashrate:   w.GetHashrateAvgGHSAll(),
			Reconnects: w.Reconnects(),
		})
		return true
	})

	slices.SortStableFunc(Workers, func(a *Worker, b *Worker) bool {
		return a.WorkerName < b.WorkerName
	})

	ctx.JSON(200, Workers)
}
