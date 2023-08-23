package main

import (
	"context"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/config"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/contractfactory"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/contractmanager"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/handlers"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/transport"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/contract"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy"
)

func main() {
	var cfg config.Config
	err := config.LoadConfig(&cfg, &os.Args)
	if err != nil {
		panic(err)
	}

	log, err := lib.NewLogger(false, cfg.Log.LevelApp, cfg.Log.LogToFile, cfg.Log.Color)
	if err != nil {
		panic(err)
	}

	schedulerLog, err := lib.NewLogger(false, cfg.Log.LevelScheduler, cfg.Log.LogToFile, cfg.Log.Color)
	if err != nil {
		panic(err)
	}

	proxyLog, err := lib.NewLogger(false, cfg.Log.LevelProxy, cfg.Log.LogToFile, cfg.Log.Color)
	if err != nil {
		panic(err)
	}

	connLog, err := lib.NewLogger(false, cfg.Log.LevelConnection, cfg.Log.LogToFile, cfg.Log.Color)
	if err != nil {
		panic(err)
	}

	var (
		HashrateCounterDefault = "ema--2.5m"
	)

	hashrateFactory := func() *hashrate.Hashrate {
		return hashrate.NewHashrateV2(
			map[string]hashrate.Counter{
				HashrateCounterDefault: hashrate.NewEma(2*time.Minute + 30*time.Second),
				"ema-10m":              hashrate.NewEma(10 * time.Minute),
				"ema-30m":              hashrate.NewEma(30 * time.Minute),
				"ema-60m":              hashrate.NewEma(1 * time.Hour),
			},
		)
	}

	ctx, cancel := context.WithCancel(context.Background())

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-shutdownChan
		log.Warnf("Received signal: %s", s)
		cancel()

		s = <-shutdownChan
		log.Warnf("Received signal: %s. Forcing exit...", s)
		os.Exit(1)
	}()

	defer func() {
		_ = log.Sync()
	}()

	destUrl, err := url.Parse(cfg.Pool.Address)
	if err != nil {
		panic(err)
	}

	var DestConnFactory = func(ctx context.Context, url *url.URL, connLogID string) (*proxy.ConnDest, error) {
		return proxy.ConnectDest(ctx, url, connLog.Named(connLogID))
	}

	alloc := allocator.NewAllocator(lib.NewCollection[*allocator.Scheduler](), log.Named("ALLOCATOR"))

	server := transport.NewTCPServer(cfg.Proxy.Address, connLog)
	server.SetConnectionHandler(func(ctx context.Context, conn net.Conn) {
		ID := conn.RemoteAddr().String()
		sourceLog := connLog.Named("[SRC] " + ID)
		sourceConn := proxy.NewSourceConn(proxy.CreateConnection(conn, ID, 5*time.Minute, 5*time.Minute, sourceLog), sourceLog)

		url := *destUrl // clones url
		currentProxy := proxy.NewProxy(ID, sourceConn, DestConnFactory, hashrateFactory, &url, proxyLog)
		scheduler := allocator.NewScheduler(currentProxy, HashrateCounterDefault, destUrl, schedulerLog)
		alloc.GetMiners().Store(scheduler)
		err = scheduler.Run(ctx)
		if err != nil {
			log.Warnf("proxy disconnected: %s", err)
		}
		alloc.GetMiners().Delete(ID)
		return
	})

	publicUrl, _ := url.Parse(cfg.Web.PublicUrl)
	hrContractFactory := contract.NewContractFactory(alloc, cfg.Hashrate.CycleDuration, hashrateFactory, log)
	cf := contractfactory.ContractFactory(hrContractFactory)
	cm := contractmanager.NewContractManager(cf, log)
	handl := handlers.NewHTTPHandler(alloc, cm, publicUrl, log)

	// create server gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.SetTrustedProxies(nil)
	r.Use(gin.Recovery())

	r.GET("/miners", handl.GetMiners)
	r.GET("/contracts", handl.GetContracts)

	r.POST("/change-dest", handl.ChangeDest)
	r.POST("/contracts", handl.CreateContract)

	go func() {
		addr := cfg.Web.Address
		log.Infof("http server is listening: %s", addr)

		err = r.Run(addr)
		if err != nil {
			panic(err)
		}
	}()

	err = server.Run(ctx)
	log.Infof("App exited due to %s", err)
}
