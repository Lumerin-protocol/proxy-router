package proxy

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"time"

	gi "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
)

const (
	CONNECTION_TIMEOUT = 10 * time.Minute
	RESPONSE_TIMEOUT   = 10 * time.Second
)

var (
	ErrConnectDest       = errors.New("failure during connecting to destination")
	ErrConnectSource     = errors.New("failure during source connection")
	ErrHandshakeDest     = errors.New("failure during handshake with destination")
	ErrHandshakeSource   = errors.New("failure during handshake with source")
	ErrProxy             = errors.New("proxy error")
	ErrNotAuthorizedPool = errors.New("not authorized in the pool")
	ErrChangeDest        = errors.New("destination change error")
)

type Proxy struct {
	// config
	ID      string
	destURL *url.URL // destination URL

	// destWorkerName string
	// submitErrLimit int
	// onFault        func(context.Context) // called when proxy becomes faulty (e.g. when submit error limit is reached

	// state
	destToSourceStartSignal chan struct{}      // signal to start reading from destination
	sourceHR                *hashrate.Hashrate // hashrate of the source validated by the proxy
	destHR                  *hashrate.Hashrate // hashrate of the destination validated by the destination
	pipe                    *Pipe
	cancelRun               context.CancelFunc // cancels Run() task
	setDestLock             sync.Mutex         // mutex to protect SetDest() from concurrent calls

	// deps
	source        *ConnSource                // initiator of the communication, miner
	dest          *ConnDest                  // receiver of the communication, pool
	destMap       *lib.Collection[*ConnDest] // map of all available destinations (pools) currently connected to the single source (miner)
	onSubmit      HashrateCounterFunc        // callback to update contract hashrate
	onSubmitMutex sync.RWMutex               // mutex to protect onSubmit

	globalHashrate GlobalHashrateCounter // callback to update global hashrate per worker
	destFactory    DestConnFactory       // factory to create new destination connections
	log            gi.ILogger
}

// TODO: pass connection factory for destURL
func NewProxy(ID string, source *ConnSource, destFactory DestConnFactory, destURL *url.URL, log gi.ILogger) *Proxy {
	proxy := &Proxy{
		ID:          ID,
		source:      source,
		destMap:     lib.NewCollection[*ConnDest](),
		destURL:     destURL,
		destFactory: destFactory,
		log:         log,

		sourceHR: hashrate.NewHashrate(),
		destHR:   hashrate.NewHashrate(),
		onSubmit: nil,
		// globalHashrate:          hashrate.NewHashrate(),
	}

	return proxy
}

var (
	minerSubscribeReceived = false
	//TODO: enforce message order validation
)

// runs proxy until handshake is done
func (p *Proxy) Connect(ctx context.Context) error {
	return NewHandlerFirstConnect(p, p.log).Connect(ctx)
}

func (p *Proxy) Run(ctx context.Context) error {
	handler := NewHandlerMining(p, p.log)

	p.pipe = NewPipe(p.source, p.dest, handler.sourceInterceptor, handler.destInterceptor, p.log)
	p.pipe.StartSourceToDest(ctx)
	p.pipe.StartDestToSource(ctx)

	ctx, cancel := context.WithCancel(ctx)
	p.cancelRun = cancel

	err := p.pipe.Run(ctx)
	if err != nil {
		p.log.Errorf("error running pipe: %s", err)

		// destination error
		if errors.Is(err, ErrDest) {
			p.log.Errorf("destination error, source %s dest %s: %s", p.source.GetID(), p.dest.GetID(), err)
			p.dest.conn.Close()
			p.destMap.Delete(p.destURL.String())
			p.dest = nil
			p.source.conn.Close()
			return err
			// TODO: reconnect to the same dest
			// return p.SetDest(ctx, p.destURL, p.onSubmit)
		}

		// source error
		if errors.Is(err, ErrSource) {
			p.log.Errorf("source error, source %s dest %s: %s", p.source.GetID(), p.dest.GetID(), err)
			// close all dest connections
			p.destMap.Range(func(dest *ConnDest) bool {
				dest.conn.Close()
				p.destMap.Delete(dest.GetID())
				return true
			})
			p.source.conn.Close()
			p.source = nil
			return err
		}

		// other errors
		return err
	}
	return nil
}

func (p *Proxy) SetDest(ctx context.Context, newDestURL *url.URL, onSubmit func(diff float64)) error {
	p.setDestLock.Lock()
	defer p.setDestLock.Unlock()

	if p.destURL.String() == newDestURL.String() {
		p.log.Infof("changing destination skipped, because it is the same as current")
		return nil
	}

	p.log.Infof("changing destination to %s", newDestURL.String())
	destChanger := NewHandlerChangeDest(p, p.destFactory, p.log)

	var newDest *ConnDest
	cachedDest, ok := p.destMap.Load(newDestURL.String())
	if ok {
		p.log.Debug("reusing dest connection %s from cache", newDestURL.String())
		err := cachedDest.AutoReadStop()
		if err != nil {
			p.log.Errorf("error stopping autoread for cached dest %s: %s", newDestURL.String(), err)
			return err
		}
		newDest = cachedDest
	} else {
		p.log.Debugf("connecting to new dest %s", newDestURL.String())
		dest, err := destChanger.connectNewDest(ctx, newDestURL)
		if err != nil {
			return err
		}
		newDest = dest
	}

	// stop source and old dest
	p.pipe.StopDestToSource()
	p.pipe.StopSourceToDest()
	p.log.Warnf("stopped source and old dest")

	// TODO: wait to stop?

	// set old dest to autoread mode
	destUrl := p.destURL.String()
	err := p.dest.AutoReadStart(ctx, func(err error) {
		if err != nil {
			p.log.Warnf("dest %s autoread exited with error %s", destUrl, err)
			p.destMap.Delete(destUrl)
			p.log.Warnf("removed old connection from the map %s", destUrl)
		}
	})
	if err != nil {
		return err
	}
	p.log.Warnf("set old dest to autoread")

	err = destChanger.resendRelevantNotifications(ctx, newDest)
	if err != nil {
		return err
	}

	p.dest = newDest
	p.destURL = newDestURL

	p.onSubmitMutex.Lock()
	p.onSubmit = onSubmit
	p.onSubmitMutex.Unlock()

	p.pipe.SetDest(newDest)
	p.log.Infof("changing dest success")

	p.pipe.StartSourceToDest(ctx)
	p.pipe.StartDestToSource(ctx)

	p.log.Debugf("resumed piping")
	return nil
}

// Getters
func (p *Proxy) GetID() string {
	return p.ID
}

func (p *Proxy) GetMinerConnectedAt() time.Time {
	return p.source.GetConnectedAt()
}

func (p *Proxy) GetDest() *url.URL {
	return p.destURL
}

func (p *Proxy) GetDestWorkerName() string {
	return p.destURL.User.Username()
}

func (p *Proxy) GetDifficulty() float64 {
	if p.dest == nil {
		return 0.0
	}
	return p.dest.GetDiff()
}

func (p *Proxy) GetHashrate() float64 {
	return float64(p.sourceHR.GetHashrateGHS())
}

func (p *Proxy) GetConnectedAt() time.Time {
	return p.source.GetConnectedAt()
}

func (p *Proxy) GetSourceWorkerName() string {
	return p.source.GetWorkerName()
}

func (p *Proxy) GetStats() interface{} {
	return p.source.GetStats()
}
