package handlers

import (
	"context"
	"errors"
	"net"
	"net/url"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/transport"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy"
)

func NewTCPHandler(
	log, connLog, proxyLog, schedulerLog interfaces.ILogger,
	notPropagateWorkerName bool, minerShareTimeout time.Duration, minerVettingShares int,
	defaultDestUrl *url.URL,
	destFactory proxy.DestConnFactory,
	hashrateFactory proxy.HashrateFactory,
	globalHashrate *hashrate.GlobalHashrate,
	hashrateCounterDefault string,
	alloc *allocator.Allocator,
) transport.Handler {
	return func(ctx context.Context, conn net.Conn) {
		ID := conn.RemoteAddr().String()
		sourceLog := connLog.Named("[SRC] " + ID)

		stratumConn := proxy.CreateConnection(conn, ID, minerShareTimeout, 10*time.Minute, sourceLog)
		defer stratumConn.Close()

		sourceConn := proxy.NewSourceConn(stratumConn, sourceLog)

		url := lib.CopyURL(defaultDestUrl) // clones url
		prx := proxy.NewProxy(ID, sourceConn, destFactory, hashrateFactory, globalHashrate, url, notPropagateWorkerName, proxyLog)
		scheduler := allocator.NewScheduler(prx, hashrateCounterDefault, url, minerVettingShares, hashrateFactory, schedulerLog)
		alloc.GetMiners().Store(scheduler)

		err := scheduler.Run(ctx)
		if err != nil {
			var logFunc func(template string, args ...interface{})
			if errors.Is(err, proxy.ErrNotStratum) {
				logFunc = connLog.Debugf
			} else if errors.Is(err, context.Canceled) {
				logFunc = connLog.Infof
			} else {
				logFunc = connLog.Errorf
			}
			logFunc("proxy disconnected: %s %s", err, ID)
		}

		alloc.GetMiners().Delete(ID)
		return
	}
}
