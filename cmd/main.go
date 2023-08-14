package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/contractfactory"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/contractmanager"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/handlers"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/transport"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/contract"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy"
)

func main() {
	appLogLevel := "debug"
	proxyLogLevel := "debug"
	connectionLogLevel := "debug"

	log, err := lib.NewLogger(false, appLogLevel, true, true)
	if err != nil {
		panic(err)
	}

	proxyLog, err := lib.NewLogger(false, proxyLogLevel, true, true)
	if err != nil {
		panic(err)
	}

	connLog, err := lib.NewLogger(false, connectionLogLevel, true, true)
	if err != nil {
		panic(err)
	}

	destUrl, _ := url.Parse("tcp://shev8.local:anything123@stratum.slushpool.com:3333")
	// destUrl, _ := url.Parse("tcp://shev8.local:anything123@0.0.0.0:3001")

	var DestConnFactory = func(ctx context.Context, url *url.URL) (*proxy.ConnDest, error) {
		return proxy.ConnectDest(ctx, url, connLog)
	}

	alloc := allocator.NewAllocator(lib.NewCollection[*allocator.Scheduler](), 10*time.Minute)

	server := transport.NewTCPServer("0.0.0.0:3333", connLog)
	server.SetConnectionHandler(func(ctx context.Context, conn net.Conn) {
		sourceLog := connLog.Named("[SRC] " + conn.RemoteAddr().String())
		sourceConn := proxy.NewSourceConn(proxy.CreateConnection(conn, &url.URL{}, 10*time.Minute, 10*time.Minute, sourceLog), sourceLog)

		url := *destUrl // clones url
		currentProxy := proxy.NewProxy(conn.RemoteAddr().String(), sourceConn, DestConnFactory, &url, proxyLog)
		scheduler := allocator.NewScheduler(currentProxy, destUrl, log)
		alloc.GetMiners().Store(scheduler)
		err = scheduler.Run(ctx)
		if err != nil {
			log.Error("proxy disconnected: ", err)
			return
		}
	})

	publicUrl, _ := url.Parse("http://localhost:3001")
	hrContractAllocator := allocator.NewAllocator(lib.NewCollection[*allocator.Scheduler](), 10*time.Minute)
	hrContractFactory := contract.NewContractFactory(hrContractAllocator, log)
	cf := contractfactory.ContractFactory(hrContractFactory)
	cm := contractmanager.NewContractManager(cf, log)
	handl := handlers.NewHTTPHandler(alloc, cm, publicUrl, log)

	// create server gin
	// gin.SetMode(gin.DebugMode)
	r := gin.New()
	r.SetTrustedProxies(nil)
	r.Use(gin.Recovery())

	r.GET("/miners", handl.GetMiners)
	r.GET("/contracts", handl.GetContracts)

	r.POST("/change-dest", handl.ChangeDest)
	r.POST("/contracts", handl.CreateContract)

	go func() {
		httpPort := 3001
		log.Infof("http server is listening: http://localhost:%d", httpPort)

		err = r.Run(fmt.Sprintf(":%d", httpPort))
		if err != nil {
			panic(err)
		}
	}()

	err = server.Run(context.Background())
	if err != nil {
		panic(err)
	}

}
