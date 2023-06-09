package main

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/handlers"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/transport"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy"
)

func main() {
	appLogLevel := "debug"
	proxyLogLevel := "debug"
	connectionLogLevel := "info"

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

	var DestConnFactory = func(ctx context.Context, url *url.URL) (*proxy.DestConn, error) {
		return proxy.ConnectDest(ctx, url, connLog)
	}

	var currentProxy proxy.Proxy

	server := transport.NewTCPServer("0.0.0.0:3000", connLog)
	server.SetConnectionHandler(func(ctx context.Context, conn net.Conn) {
		sourceLog := connLog.Named("[SRC] " + conn.RemoteAddr().String())
		sourceConn := proxy.NewSourceConn(proxy.NewConnection(conn, &url.URL{}, 10*time.Minute, time.Now(), sourceLog), sourceLog)

		currentProxy = *proxy.NewProxy(conn.RemoteAddr().String(), sourceConn, DestConnFactory, destUrl, proxyLog)
		err = currentProxy.Run(ctx)
		if err != nil {
			log.Error("proxy disconnected: ", err)
			return
		}
	})

	handl := handlers.NewHTTPHandler(&currentProxy, log)

	// create server gin
	// gin.SetMode(gin.DebugMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/change-dest", handl.ChangeDest)

	go func() {
		err = r.Run(":3001")
		if err != nil {
			panic(err)
		}
	}()

	err = server.Run(context.Background())
	if err != nil {
		panic(err)
	}

}
