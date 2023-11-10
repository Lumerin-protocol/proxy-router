package httphandlers

import (
	"context"
	"net/url"
	"time"

	"net/http/pprof"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/config"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/contractmanager"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	hr "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
)

type Proxy interface {
	SetDest(ctx context.Context, newDestURL *url.URL, onSubmit func(diff float64)) error
}

type ContractFactory func(contractData *hashrate.Terms) (resources.Contract, error)

type HTTPHandler struct {
	globalHashrate         *hr.GlobalHashrate
	allocator              *allocator.Allocator
	contractManager        *contractmanager.ContractManager
	cycleDuration          time.Duration
	hashrateCounterDefault string
	publicUrl              *url.URL
	pubKey                 string
	log                    interfaces.ILogger
}

func NewHTTPHandler(allocator *allocator.Allocator, contractManager *contractmanager.ContractManager, globalHashrate *hr.GlobalHashrate, publicUrl *url.URL, hashrateCounter string, cycleDuration time.Duration, log interfaces.ILogger) *gin.Engine {
	handl := &HTTPHandler{
		allocator:              allocator,
		contractManager:        contractManager,
		globalHashrate:         globalHashrate,
		publicUrl:              publicUrl,
		hashrateCounterDefault: hashrateCounter,
		cycleDuration:          cycleDuration,
		log:                    log,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.GET("/healthcheck", handl.HealthCheck)
	r.GET("/miners", handl.GetMiners)
	r.GET("/contracts", handl.GetContracts)
	r.GET("/contracts/:ID", handl.GetContract)
	r.GET("/contracts/:ID/logs", handl.GetDeliveryLogs)
	r.GET("/workers", handl.GetWorkers)

	r.POST("/change-dest", handl.ChangeDest)
	r.POST("/contracts", handl.CreateContract)

	r.Any("/debug/pprof/*action", gin.WrapF(pprof.Index))

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
