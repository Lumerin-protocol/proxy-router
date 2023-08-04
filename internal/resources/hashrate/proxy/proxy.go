package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	gi "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	i "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/interfaces"
	m "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
	sm "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/validator"
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
	unansweredMsg           sync.WaitGroup     // number of unanswered messages from the source
	destToSourceStartSignal chan struct{}      // signal to start reading from destination
	sourceHR                *hashrate.Hashrate // hashrate of the source validated by the proxy
	destHR                  *hashrate.Hashrate // hashrate of the destination validated by the destination
	handshakeDoneSignal     chan struct{}      // signals when first handshake is done, after miner's first connection to the default pool
	cancelRun               context.CancelFunc // cancels Run() task

	// deps
	source        *ConnSource          // initiator of the communication, miner
	dest          *ConnDest            // receiver of the communication, pool
	destMap       map[string]*ConnDest // map of all available destinations (pools) currently connected to the single source (miner)
	onSubmit      HashrateCounterFunc  // callback to update contract hashrate
	onSubmitMutex sync.RWMutex         // mutex to protect onSubmit

	globalHashrate GlobalHashrateCounter // callback to update global hashrate per worker
	destFactory    DestConnFactory       // factory to create new destination connections
	log            gi.ILogger

	pipe                *Pipe
	handshakePipe       *pipeSync
	handshakePipeTsk    *lib.Task
	cancelHandshakePipe context.CancelFunc
}

// TODO: pass connection factory for destURL
func NewProxy(ID string, source *ConnSource, destFactory DestConnFactory, destURL *url.URL, log gi.ILogger) *Proxy {
	destMap := make(map[string]*ConnDest)

	proxy := &Proxy{
		ID:          ID,
		source:      source,
		destMap:     destMap,
		destURL:     destURL,
		destFactory: destFactory,
		log:         log,

		destToSourceStartSignal: make(chan struct{}),
		sourceHR:                hashrate.NewHashrate(),
		destHR:                  hashrate.NewHashrate(),
		onSubmit:                nil,
		handshakeDoneSignal:     make(chan struct{}),
		// globalHashrate:          hashrate.NewHashrate(),
		// test
	}

	return proxy
}

var (
	minerSubscribeReceived = false
	//TODO: enforce message order validation
)

// runs proxy until handshake is done
func (p *Proxy) Connect(ctx context.Context) error {
	return p.ConnectHandshake(ctx)
}

func (p *Proxy) Run(ctx context.Context) error {
	p.pipe = NewPipe(p.source, p.dest, p.sourceInterceptor, p.destInterceptor, p.log)
	p.pipe.StartSourceToDest(ctx)
	p.pipe.StartDestToSource(ctx)

	ctx, cancel := context.WithCancel(ctx)
	p.cancelRun = cancel

	err := p.pipe.Run(ctx)
	if err != nil {
		p.log.Errorf("error running pipe: %s", err)
		return err
	}
	return nil
}

func (p *Proxy) ConnectHandshake(ctx context.Context) error {
	p.handshakePipe = NewPipeSync(p.source, p.dest, p.firstConnectHandleSource, p.firstConnectHandleDest)
	p.handshakePipeTsk = lib.NewTask(p.handshakePipe)
	handshakeCtx, handshakeCancel := context.WithCancel(ctx)
	p.cancelHandshakePipe = handshakeCancel
	p.handshakePipeTsk.Start(handshakeCtx)
	<-p.handshakePipeTsk.Done()

	if errors.Is(p.handshakePipeTsk.Err(), context.Canceled) && ctx.Err() == nil {
		return nil
	}

	return p.handshakePipeTsk.Err()
}

func (p *Proxy) firstConnectHandleSource(ctx context.Context, msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error) {
	switch msgTyped := msg.(type) {
	case *m.MiningConfigure:
		return nil, p.onMiningConfigure(ctx, msgTyped)

	case *m.MiningSubscribe:
		return nil, p.onMiningSubscribe(ctx, msgTyped)

	case *m.MiningAuthorize:
		return nil, p.onMiningAuthorize(ctx, msgTyped)

	case *m.MiningSubmit:
		return nil, fmt.Errorf("unexpected handshake message from source: %s", string(msg.Serialize()))

	default:
		p.log.Warnf("unknown handshake message from source: %s", string(msg.Serialize()))
		// todo: maybe just return message, so pipe will write it
		return nil, p.dest.Write(context.Background(), msgTyped)
	}
}

func (p *Proxy) firstConnectHandleDest(ctx context.Context, msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error) {
	var msgOut i.MiningMessageGeneric

	switch typed := msg.(type) {
	case *sm.MiningNotify:
		msgOut = typed

	case *sm.MiningSetDifficulty:
		msgOut = typed

	case *sm.MiningSetExtranonce:
		msgOut = nil

	case *sm.MiningSetVersionMask:
		msgOut = nil // sent manually

	// TODO: handle multiversion
	case *sm.MiningResult:
		msgOut = typed

	default:
		p.log.Warnf("unknown message from dest: %s", string(typed.Serialize()))
		msgOut = typed
	}

	if msgOut != nil {
		// TODO: maybe just return message, so pipe will write it, or keep it for visibility
		return nil, p.source.Write(ctx, msgOut)
	}

	return nil, nil
}

func (p *Proxy) SetDest(ctx context.Context, newDestURL *url.URL, onSubmit func(diff float64)) error {
	if p.destURL.String() == newDestURL.String() {
		p.log.Infof("changing destination skipped, because it is the same as current")
		return nil
	}

	p.log.Infof("changing destination to %s", newDestURL.String())
	var newDest *ConnDest

	cachedDest, ok := p.destMap[newDestURL.String()]
	if ok {
		p.log.Debug("reusing dest connection %s from cache", newDestURL.String())
		newDest = cachedDest
	} else {
		p.log.Debugf("connecting to new dest %s", newDestURL.String())
		dest, err := p.connectNewDest(ctx, newDestURL)
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
	oldDestAutoReadTask := lib.NewTaskFunc(p.dest.AutoRead)
	oldDestAutoReadTask.Start(ctx)
	go func() {
		urlString := p.destURL.String()
		<-oldDestAutoReadTask.Done()
		p.log.Warnf("dest %s autoread exited with error %s", urlString, oldDestAutoReadTask.Err())
		delete(p.destMap, urlString)
	}()

	p.log.Warnf("set old dest to autoread")

	// resend relevant notifications to the miner
	// 1. SET_VERSION_MASK
	_, versionMask := newDest.GetVersionRolling()
	err := p.source.Write(ctx, m.NewMiningSetVersionMask(versionMask))
	if err != nil {
		return lib.WrapError(ErrChangeDest, err)
	}
	p.log.Warnf("set version mask sent")

	job, ok := newDest.GetLatestJob()
	if !ok {
		return lib.WrapError(ErrChangeDest, errors.New("no job available"))
	}

	// 2. SET_EXTRANONCE
	err = p.source.Write(ctx, m.NewMiningSetExtranonce(job.GetExtraNonce1(), job.GetExtraNonce2Size()))
	if err != nil {
		return lib.WrapError(ErrChangeDest, err)
	}
	p.source.SetExtraNonce(job.GetExtraNonce1(), job.GetExtraNonce2Size())
	p.log.Warnf("extranonce sent")

	// 3. SET_DIFFICULTY
	err = p.source.Write(ctx, m.NewMiningSetDifficulty(job.GetDiff()))
	if err != nil {
		return lib.WrapError(ErrChangeDest, err)
	}
	p.log.Warnf("set difficulty sent")

	// 4. NOTIFY
	msg := job.GetNotify()
	msg.SetCleanJobs(true)

	err = p.source.Write(ctx, msg)
	if err != nil {
		return lib.WrapError(ErrChangeDest, err)
	}
	p.log.Warnf("notify sent")

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

func (p *Proxy) connectNewDest(ctx context.Context, newDestURL *url.URL) (*ConnDest, error) {
	newDest, err := p.destFactory(ctx, newDestURL)
	if err != nil {
		return nil, lib.WrapError(ErrConnectDest, err)
	}

	p.log.Debugf("new dest created")

	autoReadTask := lib.NewTaskFunc(newDest.AutoRead)
	autoReadTask.Start(ctx)

	p.log.Debugf("dest autoread started")

	handshakeTask := lib.NewTaskFunc(func(ctx context.Context) error {
		user := newDestURL.User.Username()
		pwd, _ := newDestURL.User.Password()
		return p.destHandshake(ctx, newDest, user, pwd)
	})

	handshakeTask.Start(ctx)

	select {
	case <-autoReadTask.Done():
		// if newDestRunTask finished first there was reading error
		return nil, lib.WrapError(ErrConnectDest, autoReadTask.Err())
	case <-handshakeTask.Done():
	}

	if handshakeTask.Err() != nil {
		return nil, lib.WrapError(ErrConnectDest, handshakeTask.Err())
	}
	p.log.Debugf("new dest connected")

	// stops temporary reading from newDest
	<-autoReadTask.Stop()
	p.log.Debugf("stopped new dest")
	return newDest, nil
}

// destHandshake performs handshake with the new dest when there is a dest that already connected
func (p *Proxy) destHandshake(ctx context.Context, newDest *ConnDest, user string, pwd string) error {
	msgID := 1

	// 1. MINING.CONFIGURE
	// if miner has version mask enabled, send it to the pool
	if p.source.GetNegotiatedVersionRollingMask() != "" {
		// using the same version mask as the miner negotiated during the prev connection
		cfgMsg := m.NewMiningConfigure(msgID, nil)
		_, minBits := p.source.GetVersionRolling()
		cfgMsg.SetVersionRolling(p.source.GetNegotiatedVersionRollingMask(), minBits)

		res, err := newDest.WriteAwaitRes(ctx, cfgMsg)
		if err != nil {
			return lib.WrapError(ErrConnectDest, err)
		}

		cfgRes, err := m.ToMiningConfigureResult(res.(*m.MiningResult))
		if err != nil {
			return err
		}
		if cfgRes.IsError() {
			return fmt.Errorf("pool returned error: %s", cfgRes.GetError())
		}

		if cfgRes.GetVersionRollingMask() != p.source.GetNegotiatedVersionRollingMask() {
			// what to do if pool has different mask
			// TODO: consider sending set_version_mask to the pool? https://en.bitcoin.it/wiki/BIP_0310
			return fmt.Errorf("pool returned different version rolling mask: %s", cfgRes.GetVersionRollingMask())
		}

		newDest.SetVersionRolling(true, cfgRes.GetVersionRollingMask())
		p.log.Debugf("configure result received")
	}

	// 2. MINING.SUBSCRIBE
	msgID++
	res, err := newDest.WriteAwaitRes(ctx, m.NewMiningSubscribe(msgID, "stratum-proxy", "1.0.0"))
	if err != nil {
		return lib.WrapError(ErrConnectDest, err)
	}
	subRes, err := m.ToMiningSubscribeResult(res.(*m.MiningResult))
	if err != nil {
		return err
	}
	if subRes.IsError() {
		return fmt.Errorf("pool returned error: %s", subRes.GetError())
	}

	newDest.SetExtraNonce(subRes.GetExtranonce())
	p.log.Debugf("subscribe result received")

	// 3. MINING.AUTHORIZE
	msgID++

	res, err = newDest.WriteAwaitRes(ctx, m.NewMiningAuthorize(msgID, user, pwd))
	if err != nil {
		return lib.WrapError(ErrConnectDest, err)
	}

	authRes := res.(*m.MiningResult)
	if authRes.IsError() {
		return lib.WrapError(ErrConnectDest, lib.WrapError(ErrNotAuthorizedPool, fmt.Errorf("%s", authRes.GetError())))
	}

	p.log.Debugf("authorize success")
	return nil
}

// sourceInterceptor is called when a message is received from the source after handshake
func (p *Proxy) sourceInterceptor(ctx context.Context, msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error) {
	switch msgTyped := msg.(type) {
	case *m.MiningSubmit:
		return p.onMiningSubmit(ctx, msgTyped)
	// errors
	case *m.MiningConfigure:
		return nil, fmt.Errorf("unexpected message from source after handshake: %s", string(msg.Serialize()))
	case *m.MiningSubscribe:
		return nil, fmt.Errorf("unexpected message from source after handshake: %s", string(msg.Serialize()))
	case *m.MiningAuthorize:
		return nil, fmt.Errorf("unexpected message from source after handshake: %s", string(msg.Serialize()))
	default:
		p.log.Warn("unknown message from source: %s", string(msg.Serialize()))
		return msg, nil
	}
}

// destInterceptor is called when a message is received from the dest after handshake
func (p *Proxy) destInterceptor(ctx context.Context, msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error) {
	switch msgTyped := msg.(type) {
	case *m.MiningSetDifficulty:
		p.log.Debugf("new diff: %.0f", msgTyped.GetDifficulty())
		return msg, nil
	case *m.MiningSetVersionMask:
		p.log.Debugf("got version mask: %s", msgTyped.GetVersionMask())
		return msg, nil
	case *m.MiningSetExtranonce:
		xn, xn2size := msgTyped.GetExtranonce()
		p.log.Debugf("got extranonce: %s %s", xn, xn2size)
		return msg, nil
	case *m.MiningNotify:
		return msg, nil
	case *m.MiningResult:
		return msg, nil
	default:
		p.log.Warn("unknown message from dest: %s", string(msg.Serialize()))
		return msg, nil
	}
}

// The following handlers are responsible for managing the initial connection of the miner to the proxy and destination.
// This step requires special handling due to the "coupled" interaction between parties, unlike the destination change process,
// where the pool connection is established first, and then the miner is switched to it. This "coupled" interaction is intentionally
// designed to enable the negotiation of the version rolling mask. It's important to note that all of these handlers require
// performing reads and writes within the same goroutine. Additionally, other response handlers (identified by message ID) must be
// called right after receiving the message. This ensures that the order of messages is deterministic. If the order of messages
// during the handshake is not enforced, there is a possibility that miners may fail, for example, if the "set_version_mask"
// message is sent to the miner before receiving the "configure" result.

func (p *Proxy) onMiningConfigure(ctx context.Context, msgTyped *m.MiningConfigure) error {
	p.source.SetVersionRolling(msgTyped.GetVersionRolling())

	destConn, err := p.destFactory(ctx, p.destURL)
	if err != nil {
		return err
	}

	p.dest = destConn
	p.handshakePipe.SetStream2(destConn)
	p.handshakePipe.StartStream2()

	err = p.dest.Write(ctx, msgTyped)
	if err != nil {
		return lib.WrapError(ErrHandshakeDest, err)
	}

	p.dest.onceResult(ctx, msgTyped.GetID(), func(res *sm.MiningResult) (msg i.MiningMessageWithID, err error) {
		configureResult, err := m.ToMiningConfigureResult(res)
		if err != nil {
			p.log.Errorf("expected MiningConfigureResult message, got %s", res.Serialize())
			return nil, err
		}

		vr, mask := configureResult.GetVersionRolling(), configureResult.GetVersionRollingMask()
		destConn.SetVersionRolling(vr, mask)
		p.source.SetNegotiatedVersionRollingMask(mask)

		configureResult.SetID(msgTyped.GetID())
		err = p.source.Write(ctx, configureResult)
		if err != nil {
			return nil, lib.WrapError(ErrHandshakeSource, err)
		}

		err = p.source.Write(ctx, m.NewMiningSetVersionMask(configureResult.GetVersionRollingMask()))
		if err != nil {
			return nil, lib.WrapError(ErrHandshakeSource, err)
		}
		return nil, nil
	})

	return nil
}

func (p *Proxy) onMiningSubscribe(ctx context.Context, msgTyped *m.MiningSubscribe) error {
	minerSubscribeReceived = true

	if p.dest == nil {
		destConn, err := p.destFactory(ctx, p.destURL)
		if err != nil {
			return err
		}

		p.dest = destConn
		p.handshakePipe.SetStream2(destConn)
		p.handshakePipe.StartStream2()
	}

	err := p.dest.Write(ctx, msgTyped)
	if err != nil {
		return lib.WrapError(ErrHandshakeDest, err)
	}

	p.dest.onceResult(ctx, msgTyped.GetID(), func(res *sm.MiningResult) (msg i.MiningMessageWithID, err error) {
		subscribeResult, err := m.ToMiningSubscribeResult(res)
		if err != nil {
			return nil, fmt.Errorf("expected MiningSubscribeResult message, got %s", res.Serialize())
		}

		p.source.SetExtraNonce(subscribeResult.GetExtranonce())
		p.dest.SetExtraNonce(subscribeResult.GetExtranonce())

		subscribeResult.SetID(msgTyped.GetID())

		err = p.source.Write(ctx, subscribeResult)
		if err != nil {
			return nil, lib.WrapError(ErrHandshakeSource, err)
		}
		return nil, nil
	})

	return nil
}

func (p *Proxy) onMiningAuthorize(ctx context.Context, msgTyped *m.MiningAuthorize) error {
	p.source.SetWorkerName(msgTyped.GetWorkerName())
	p.log = p.log.Named(msgTyped.GetWorkerName())

	msgID := msgTyped.GetID()
	if !minerSubscribeReceived {
		return lib.WrapError(ErrHandshakeSource, fmt.Errorf("MiningAuthorize received before MiningSubscribe"))
	}

	msg := m.NewMiningResultSuccess(msgID)
	err := p.source.Write(ctx, msg)
	if err != nil {
		return lib.WrapError(ErrHandshakeSource, err)
	}

	_, workerName, _ := lib.SplitUsername(msgTyped.GetWorkerName())
	lib.SetWorkerName(p.destURL, workerName)
	userName := p.destURL.User.Username()

	p.dest.SetWorkerName(userName)

	pwd, ok := p.destURL.User.Password()
	if !ok {
		pwd = ""
	}
	destAuthMsg := m.NewMiningAuthorize(msgID, userName, pwd)

	err = p.dest.Write(ctx, destAuthMsg)
	if err != nil {
		return lib.WrapError(ErrHandshakeDest, err)
	}

	p.dest.onceResult(ctx, msgID, func(res *sm.MiningResult) (msg i.MiningMessageWithID, err error) {
		if res.IsError() {
			return nil, lib.WrapError(ErrHandshakeDest, fmt.Errorf("cannot authorize in dest pool: %s", res.GetError()))
		}
		p.log.Infof("connected to destination: %s", p.destURL.String())
		p.log.Info("handshake completed")

		p.destMap[p.destURL.String()] = p.dest

		// close
		p.cancelHandshakePipe()

		return nil, nil
	})

	return nil
}

// onMiningSubmit is only called when handshake is completed. It doesn't require determinism
// in message ordering, so to improve performance we can use asynchronous pipe
func (p *Proxy) onMiningSubmit(ctx context.Context, msgTyped *m.MiningSubmit) (i.MiningMessageGeneric, error) {
	p.unansweredMsg.Add(1)

	diff, err := p.dest.ValidateAndAddShare(msgTyped)
	weAccepted := err == nil
	var res *m.MiningResult

	if err != nil {
		p.source.GetStats().WeRejectedShares++

		if errors.Is(err, validator.ErrJobNotFound) {
			res = m.NewMiningResultJobNotFound(msgTyped.GetID())
		} else if errors.Is(err, validator.ErrDuplicateShare) {
			res = m.NewMiningResultDuplicatedShare(msgTyped.GetID())
		} else if errors.Is(err, validator.ErrLowDifficulty) {
			res = m.NewMiningResultLowDifficulty(msgTyped.GetID())
		}

	} else {
		p.source.GetStats().WeAcceptedShares++
		// miner hashrate
		p.sourceHR.OnSubmit(p.dest.GetDiff())

		// contract hashrate
		p.onSubmitMutex.RLock()
		if p.onSubmit != nil {
			p.onSubmit(p.dest.GetDiff())
		}
		p.onSubmitMutex.RUnlock()

		res = m.NewMiningResultSuccess(msgTyped.GetID())
	}

	// workername hashrate
	// s.globalHashrate.OnSubmit(s.source.GetWorkerName(), s.dest.GetDiff())

	// does not wait for response from destination pool
	// TODO: implement buffering for source/dest messages
	// to avoid blocking source/dest when one of them is slow
	// and fix error handling to avoid p.cancelRun
	go func(res1 *m.MiningResult) {
		err = p.source.Write(ctx, res1)
		if err != nil {
			p.log.Error("cannot write response to miner: ", err)
			p.cancelRun()
		}

		// send and await submit response from pool
		msgTyped.SetWorkerName(p.dest.GetWorkerName())
		res, err := p.dest.WriteAwaitRes(ctx, msgTyped)
		if err != nil {
			p.log.Error("cannot write response to pool: ", err)
			p.cancelRun()
		}
		p.unansweredMsg.Done()

		if res.(*m.MiningResult).IsError() {
			if weAccepted {
				p.source.GetStats().WeAcceptedTheyRejected++
				p.dest.GetStats().WeAcceptedTheyRejected++
			}
		} else {
			if weAccepted {
				p.dest.GetStats().WeAcceptedTheyAccepted++
				p.destHR.OnSubmit(p.dest.GetDiff())
				p.log.Debugf("new submit, diff: %0.f, target: %0.f", diff, p.dest.GetDiff())
			} else {
				p.dest.GetStats().WeRejectedTheyAccepted++
				p.source.GetStats().WeRejectedTheyAccepted++
				p.log.Warnf("we rejected submit, but dest accepted, diff: %d, target: %0.f", diff, p.dest.GetDiff())
			}
		}
	}(res)

	return nil, nil
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
