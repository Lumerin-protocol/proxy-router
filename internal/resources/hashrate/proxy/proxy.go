package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"sync"
	"time"

	gi "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"go.uber.org/atomic"
)

const (
	CONNECTION_TIMEOUT = 10 * time.Minute
	RESPONSE_TIMEOUT   = 30 * time.Second
)

var (
	ErrConnectDest       = errors.New("failure during connecting to destination")
	ErrConnectSource     = errors.New("failure during source connection")
	ErrHandshakeDest     = errors.New("failure during handshake with destination")
	ErrHandshakeSource   = errors.New("failure during handshake with source")
	ErrProxy             = errors.New("proxy error")
	ErrNotAuthorizedPool = errors.New("not authorized in the pool")
	ErrChangeDest        = errors.New("destination change error")
	ErrAutoreadStarted   = errors.New("autoread already started")
)

type Proxy struct {
	// config
	ID                     string
	destURL                *atomic.Pointer[url.URL]
	notPropagateWorkerName bool
	maxCachedDests         int // maximum number of cached dests

	// state
	destToSourceStartSignal chan struct{}      // signal to start reading from destination
	hashrate                *hashrate.Hashrate // hashrate of the source validated by the proxy
	pipe                    *Pipe
	cancelRun               context.CancelFunc         // cancels Run() task
	setDestLock             sync.Mutex                 // mutex to protect SetDest() from concurrent calls
	unansweredMsg           sync.WaitGroup             // number of unanswered messages from the source
	onSubmit                HashrateCounterFunc        // callback to update contract hashrate
	onSubmitMutex           sync.RWMutex               // mutex to protect onSubmit
	destMap                 *lib.Collection[*ConnDest] // map of all available destinations (pools) currently connected to the single source (miner)
	vettingDoneCh           chan struct{}              // channel to signal that the miner has been vetted
	vettingShares           int                        // number of shares to vet the miner

	// deps
	source         *ConnSource           // initiator of the communication, miner
	dest           *ConnDest             // receiver of the communication, pool
	globalHashrate GlobalHashrateCounter // callback to update global hashrate per worker
	destFactory    DestConnFactory       // factory to create new destination connections
	log            gi.ILogger
}

func NewProxy(ID string, source *ConnSource, destFactory DestConnFactory, hashrateFactory HashrateFactory, globalHashrate GlobalHashrateCounter, destURL *url.URL, notPropagateWorkerName bool, vettingShares int, maxCachedDests int, log gi.ILogger) *Proxy {
	proxy := &Proxy{
		ID:                     ID,
		destURL:                atomic.NewPointer(destURL),
		notPropagateWorkerName: notPropagateWorkerName,
		maxCachedDests:         maxCachedDests,

		source:        source,
		destMap:       lib.NewCollection[*ConnDest](),
		destFactory:   destFactory,
		log:           log,
		vettingDoneCh: make(chan struct{}),
		vettingShares: vettingShares,

		hashrate:       hashrateFactory(),
		globalHashrate: globalHashrate,
		onSubmit:       nil,
	}

	return proxy
}

var (
	minerSubscribeReceived = false
	//TODO: enforce message order validation
)

// runs proxy until handshake is done
func (p *Proxy) Connect(ctx context.Context) error {
	err := NewHandlerFirstConnect(p, p.log).Connect(ctx)
	if err != nil {
		p.closeConnections()
		return err
	}
	return nil
}

func (p *Proxy) Run(ctx context.Context) error {
	defer p.closeConnections()
	handler := NewHandlerMining(p, p.log)
	p.pipe = NewPipe(p.source, p.dest, handler.sourceInterceptor, handler.destInterceptor, p.log)

	for {
		p.pipe.StartSourceToDest(ctx)
		p.pipe.StartDestToSource(ctx)

		ctx, cancel := context.WithCancel(ctx)
		p.cancelRun = cancel

		err := p.pipe.Run(ctx)
		p.unansweredMsg.Wait()

		if err != nil {
			// destination error
			if errors.Is(err, ErrDest) {
				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
					p.log.Warnf("destination closed the connection, dest %s", p.dest.ID())
				} else {
					p.log.Errorf("destination error, source %s dest %s: %s", p.source.GetID(), p.dest.ID(), err)
				}

				// try to reconnect to the same dest
				p.destMap.Delete(p.dest.ID())
				err := p.ConnectDest(ctx, lib.CopyURL(p.destURL.Load()))
				if err != nil {
					p.log.Errorf("error reconnecting to dest %s: %s", p.dest.ID(), err)
					return err
				}
				continue
			}

			// source error
			if errors.Is(err, ErrSource) {
				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
					p.log.Warnf("source closed the connection, source %s, err %s", p.source.GetID(), err)
				} else {
					p.log.Errorf("source connection error, source %s: %s", p.source.GetID(), err)
				}
				return err
			}

			if errors.Is(err, context.Canceled) {
				return err
			}

			p.log.Errorf("error running pipe: %s", err)

			// other errors
			return err
		}
		return nil
	}
}

func (p *Proxy) ConnectDest(ctx context.Context, newDestURL *url.URL) error {
	p.setDestLock.Lock()
	defer p.setDestLock.Unlock()

	p.log.Debugf("reconnecting to destination %s", newDestURL.String())

	destChanger := NewHandlerChangeDest(p, p.destFactory, p.log)

	newDest, err := destChanger.connectNewDest(ctx, newDestURL)
	if err != nil {
		return err
	}

	err = destChanger.resendRelevantNotifications(ctx, newDest)
	if err != nil {
		return err
	}

	p.dest = newDest
	p.destURL.Store(newDestURL)
	p.destMap.Store(newDest)

	p.pipe.SetDest(newDest)

	p.pipe.StartSourceToDest(ctx)
	p.pipe.StartDestToSource(ctx)

	p.log.Infof("destination reconnected %s", newDestURL.String())

	return nil
}

func (p *Proxy) SetDest(ctx context.Context, newDestURL *url.URL, onSubmit func(diff float64)) error {
	return p.setDest(ctx, newDestURL, onSubmit, true)
}

func (p *Proxy) SetDestWithoutAutoread(ctx context.Context, newDestURL *url.URL, onSubmit func(diff float64)) error {
	return p.setDest(ctx, newDestURL, onSubmit, false)
}

func (p *Proxy) setDest(ctx context.Context, newDestURL *url.URL, onSubmit func(diff float64), autoReadCurrentDest bool) error {
	p.setDestLock.Lock()
	defer p.setDestLock.Unlock()

	if p.destURL.String() == newDestURL.String() {
		p.log.Debugf("changing destination skipped, because it is the same as current")
		return nil
	}

	p.log.Debugf("changing destination to %s", newDestURL.String())
	destChanger := NewHandlerChangeDest(p, p.destFactory, p.log)

	var newDest *ConnDest
	cachedDest, ok := p.destMap.Load(newDestURL.String())
	if ok {
		p.log.Infof("reusing connection from cache %s", newDestURL.String())
		// limit waiting time, disconnect if not answered in time
		p.unansweredMsg.Wait()
		err := cachedDest.AutoReadStop()
		if err != nil {
			p.log.Errorf("error stopping autoread for cached dest %s: %s", newDestURL.String(), err)
			return err
		}
		cachedDest.ResetIdleCloseTimers()
		newDest = cachedDest
	} else {
		p.log.Infof("connecting to new dest %s", newDestURL.String())
		dest, err := destChanger.connectNewDest(ctx, newDestURL)
		if err != nil {
			return err
		}

		p.unansweredMsg.Wait()
		newDest = dest
	}

	// stop source and old dest
	<-p.pipe.StopDestToSource()
	<-p.pipe.StopSourceToDest()
	p.log.Debugf("stopped source and old dest")

	// TODO: wait to stop?

	// set old dest to autoread mode
	if autoReadCurrentDest {
		destUrl := p.destURL.String()
		dest := p.dest
		ok := dest.AutoReadStart(ctx, func(err error) {
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					p.log.Infof("cached connection closed: %s %s", destUrl, err.Error())
				} else {
					p.log.Warnf("autoread exited with error %s", err)
					err := dest.conn.Close()
					if err != nil {
						p.log.Warnf("error closing dest %s: %s", destUrl, err)
					}
				}
			}

			p.destMap.Delete(destUrl)
		})
		if !ok {
			p.destMap.Delete(dest.ID())
			p.log.Errorf("%s dest ID %s", ErrAutoreadStarted, dest.ID())
		} else {
			p.log.Debugf("set old dest to autoread")
		}
	}

	err := destChanger.resendRelevantNotifications(ctx, newDest)
	if err != nil {
		return err
	}

	if p.destMap.Len() >= p.maxCachedDests {
		p.closeOldestConn()
	}

	p.dest = newDest
	p.destURL.Store(newDestURL)
	p.destMap.Store(newDest)

	p.onSubmitMutex.Lock()
	p.onSubmit = onSubmit
	p.onSubmitMutex.Unlock()

	p.pipe.SetDest(newDest)

	p.pipe.StartSourceToDest(ctx)
	p.pipe.StartDestToSource(ctx)

	p.log.Infof("destination changed to %s", newDestURL.String())
	return nil
}

func (p *Proxy) closeOldestConn() {
	p.log.Debugf("dest map size %d exceeds max cached dests %d, closing oldest dest", p.destMap.Len(), p.maxCachedDests)
	var oldestDest *ConnDest

	p.destMap.Range(func(dest *ConnDest) bool {
		if oldestDest == nil {
			oldestDest = dest
			return true
		}

		if oldestDest.GetIdleCloseAt().After(dest.GetIdleCloseAt()) {
			oldestDest = dest
		}
		return true
	})
	oldestDest.conn.Close()
	p.destMap.Delete(oldestDest.ID())
}

func (p *Proxy) closeConnections() {
	if p.dest != nil {
		p.dest.conn.Close()
	}

	p.destMap.Range(func(dest *ConnDest) bool {
		dest.conn.Close()
		p.destMap.Delete(dest.ID())
		return true
	})
}

func (p *Proxy) GetDestByJobID(jobID string) *ConnDest {
	var dest *ConnDest

	p.destMap.Range(func(d *ConnDest) bool {
		if d.HasJob(jobID) {
			dest = d
			return false
		}
		return true
	})

	return dest
}

// Getters
func (p *Proxy) GetID() string {
	return p.ID
}

func (p *Proxy) GetMinerConnectedAt() time.Time {
	return p.source.GetConnectedAt()
}

func (p *Proxy) GetDest() *url.URL {
	return p.destURL.Load()
}

func (p *Proxy) GetDestWorkerName() string {
	return p.destURL.Load().User.Username()
}

func (p *Proxy) GetDifficulty() float64 {
	if p.dest == nil {
		return 0.0
	}
	return p.dest.GetDiff()
}

func (p *Proxy) GetHashrate() Hashrate {
	return p.hashrate
}

func (p *Proxy) GetConnectedAt() time.Time {
	return p.source.GetConnectedAt()
}

func (p *Proxy) GetSourceWorkerName() string {
	return p.source.GetUserName()
}

func (p *Proxy) GetStats() map[string]int {
	return p.source.GetStats().GetStatsMap()
}

func (p *Proxy) GetDestConns() *map[string]string {
	var destConns = make(map[string]string)
	p.destMap.Range(func(dest *ConnDest) bool {
		destConns[dest.ID()] = dest.GetIdleCloseAt().Sub(time.Now()).Round(time.Second).String()
		return true
	})
	return &destConns
}

func (p *Proxy) VettingDone() <-chan struct{} {
	return p.vettingDoneCh
}

func (p *Proxy) IsVetting() bool {
	return p.GetHashrate().GetTotalShares() < p.vettingShares
}
