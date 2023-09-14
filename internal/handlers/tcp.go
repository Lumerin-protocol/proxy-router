package handlers

import (
	"context"
	"net"
	"net/url"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/transport"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy"
)

// cfg.Miner.ShareTimeout

func NewTCPHandler(
	log, connLog, proxyLog, schedulerLog interfaces.ILogger,
	minerShareTimeout, minerVettingDuration time.Duration,
	defaultDestUrl *url.URL,
	destConnFactory proxy.DestConnFactory,
	hashrateFactory proxy.HashrateFactory,
	globalHashrate *hashrate.GlobalHashrate,
	hashrateCounterDefault string,
	alloc *allocator.Allocator,
) transport.Handler {
	return func(ctx context.Context, conn net.Conn) {
		ID := conn.RemoteAddr().String()
		sourceLog := connLog.Named("[SRC] " + ID)

		stratumConn := proxy.CreateConnection(conn, ID, minerShareTimeout, 5*time.Minute, sourceLog)
		defer stratumConn.Close()

		sourceConn := proxy.NewSourceConn(stratumConn, sourceLog)

		url := *defaultDestUrl // clones url
		proxy := proxy.NewProxy(ID, sourceConn, destConnFactory, hashrateFactory, globalHashrate, &url, proxyLog)
		scheduler := allocator.NewScheduler(proxy, hashrateCounterDefault, &url, minerVettingDuration, schedulerLog)
		alloc.GetMiners().Store(scheduler)

		err := scheduler.Run(ctx)
		if err != nil {
			log.Warnf("proxy disconnected: %s", err)
		}

		alloc.GetMiners().Delete(ID)
		return
	}
}
