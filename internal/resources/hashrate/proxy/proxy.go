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

type ResultHandler = func(a *m.MiningResult) (msg i.MiningMessageToPool, err error)

type Proxy struct {
	// config
	ID             string
	destWorkerName string
	submitErrLimit int
	onFault        func(context.Context) // called when proxy becomes faulty (e.g. when submit error limit is reached
	destURL        *url.URL              // destination URL

	// state
	unansweredMsg           sync.WaitGroup     // number of unanswered messages from the source
	destToSourceStartSignal chan struct{}      // signal to start reading from destination
	sourceHR                *hashrate.Hashrate // hashrate of the source validated by the proxy
	destHR                  *hashrate.Hashrate // hashrate of the destination validated by the destination

	// deps
	source         *ConnSource           // initiator of the communication, miner
	dest           *ConnDest             // receiver of the communication, pool
	destMap        map[string]*ConnDest  // map of all available destinations (pools) currently connected to the single source (miner)
	onSubmit       HashrateCounter       // callback to update contract hashrate
	onSubmitMutex  sync.RWMutex          // mutex to protect onSubmit
	globalHashrate GlobalHashrateCounter // callback to update global hashrate per worker
	destFactory    DestConnFactory       // factory to create new destination connections
	log            gi.ILogger

	pipe *Pipe
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
		onSubmit:                hashrate.NewHashrate(),
		// globalHashrate:          hashrate.NewHashrate(),
	}

	pipe := NewPipe(source, nil, proxy.sourceInterceptor, proxy.destInterceptor, log)
	proxy.pipe = pipe

	return proxy
}

var (
	minerSubscribeReceived = false
	//TODO: enforce message order validation
)

func (p *Proxy) Run(ctx context.Context) error {
	p.pipe.StartSourceToDest(ctx)
	err := p.pipe.Run(ctx)
	if err != nil {
		p.log.Errorf("error running pipe: %s", err)
		return err
	}
	return nil
}

func (p *Proxy) ChangeDest(ctx context.Context, newDestURL *url.URL) error {
	p.log.Warnf("changing destination to %s", newDestURL.String())

	dest, ok := p.destMap[newDestURL.String()]
	if ok {
		p.log.Debug("reusing dest connection %s from cache", newDestURL.String())
	} else {
		p.log.Debug("connecting to new dest", newDestURL.String())
		newDest, err := p.connectNewDest(ctx, newDestURL)
		if err != nil {
			return err
		}
		dest = newDest
	}

	// stop source and old dest
	p.pipe.StopDestToSource()
	p.pipe.StopSourceToDest()

	p.log.Warnf("stopped source and old dest")

	// TODO: wait to stop?

	// set old dest to autoread mode
	p.dest.AutoReadStart()
	p.log.Warnf("set old dest to autoread")

	// resend relevant notifications to the miner
	// 1. SET_EXTRANONCE
	err := p.source.Write(ctx, m.NewMiningSetExtranonce(dest.GetExtraNonce()))
	if err != nil {
		return lib.WrapError(ErrChangeDest, err)
	}
	p.source.SetExtraNonce(dest.GetExtraNonce())
	p.log.Warnf("extranonce sent")

	// 2. SET_DIFFICULTY
	err = p.source.Write(ctx, m.NewMiningSetDifficulty(p.dest.GetDiff()))
	if err != nil {
		return lib.WrapError(ErrChangeDest, err)
	}
	p.log.Warnf("set difficulty sent")

	// TODO: 3. SET_VERSION_MASK

	// 4. NOTIFY
	msg, ok := dest.notifyMsgs.At(-1)
	if ok {
		msg = msg.Copy()
		msg.SetCleanJobs(true)
		err = p.source.Write(ctx, msg)
		if err != nil {
			return lib.WrapError(ErrChangeDest, err)
		}
		p.log.Warnf("notify sent")
	} else {
		p.log.Warnf("no notify msg found")
	}

	p.pipe.StopDestToSource()
	p.pipe.StopSourceToDest()

	p.dest = dest
	p.destURL = newDestURL

	p.pipe.SetDest(dest)
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

	newDestRunTask := lib.NewTaskFunc(newDest.Run)
	newDest.AutoReadStart()

	handshakeTask := lib.NewTaskFunc(func(ctx context.Context) error {
		user := newDestURL.User.Username()
		pwd, _ := newDestURL.User.Password()
		return p.destHandshake(ctx, newDest, user, pwd)
	})

	select {
	case <-newDestRunTask.Done():
		// if newDestRunTask finished first there was reading error
		return nil, lib.WrapError(ErrConnectDest, newDestRunTask.Err())
	case <-handshakeTask.Done():
	}

	if handshakeTask.Err() != nil {
		return nil, lib.WrapError(ErrConnectDest, handshakeTask.Err())
	}
	p.log.Debugf("new dest connected")

	// stops temporary reading from newDest
	<-newDestRunTask.Stop()
	p.log.Debugf("stopped new dest")
	return newDest, nil
}

func (p *Proxy) destHandshake(ctx context.Context, newDest *ConnDest, user string, pwd string) error {
	msgID := 1
	p.log.Debugf("new dest autoread started")

	// 1. MINING.CONFIGURE
	// if miner has version mask enabled, send it to the pool
	if p.source.GetNegotiatedVersionRollingMask() != "" {
		// using the same version mask as the miner negotiated during the prev connection
		cfgMsg := m.NewMiningConfigure(msgID, nil)
		_, minBits := p.source.GetVersionRolling()
		cfgMsg.SetVersionRolling(p.source.GetNegotiatedVersionRollingMask(), minBits)

		err := newDest.Write(ctx, m.NewMiningConfigure(msgID, nil))
		if err != nil {
			return lib.WrapError(ErrConnectDest, err)
		}
		p.log.Debugf("configure sent")

		<-newDest.onceResult(ctx, msgID, func(msg *m.MiningResult) (i.MiningMessageToPool, error) {
			cfgRes, err := m.ToMiningConfigureResult(msg)
			if err != nil {
				return nil, err
			}
			if cfgRes.IsError() {
				return nil, fmt.Errorf("pool returned error: %s", cfgRes.GetError())
			}
			if cfgRes.GetVersionRollingMask() != p.source.GetNegotiatedVersionRollingMask() {
				// what to do if pool has different mask
				return nil, fmt.Errorf("pool returned different version rolling mask: %s", cfgRes.GetVersionRollingMask())
				// TODO: consider sending set_version_mask to the pool? https://en.bitcoin.it/wiki/BIP_0310
			}
			newDest.SetVersionRolling(true, cfgRes.GetVersionRollingMask())
			return nil, nil
		})

		p.log.Debugf("configure result received")
	}

	// 2. MINING.SUBSCRIBE
	msgID++
	err := newDest.Write(ctx, m.NewMiningSubscribe(msgID, "stratum-proxy", "1.0.0"))
	if err != nil {
		return lib.WrapError(ErrConnectDest, err)
	}
	p.log.Debugf("subscribe sent")

	<-newDest.onceResult(ctx, msgID, func(msg *m.MiningResult) (i.MiningMessageToPool, error) {
		subRes, err := m.ToMiningSubscribeResult(msg)
		if err != nil {
			return nil, err
		}
		if subRes.IsError() {
			return nil, fmt.Errorf("pool returned error: %s", subRes.GetError())
		}

		newDest.SetExtraNonce(subRes.GetExtranonce())

		return nil, nil
	})
	p.log.Debugf("subscribe result received")

	// 3. MINING.AUTHORIZE
	msgID++
	err = newDest.Write(ctx, m.NewMiningAuthorize(msgID, user, pwd))
	if err != nil {
		return lib.WrapError(ErrConnectDest, err)
	}
	p.log.Debugf("authorize sent")

	<-newDest.onceResult(ctx, msgID, func(msg *m.MiningResult) (i.MiningMessageToPool, error) {
		if msg.IsError() {
			return nil, lib.WrapError(ErrConnectDest, lib.WrapError(ErrNotAuthorizedPool, fmt.Errorf("%s", msg.GetError())))
		}
		return nil, nil
	})
	p.log.Debugf("authorize success")

	return nil
}

func (p *Proxy) destInterceptor(msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error) {
	switch msgTyped := msg.(type) {
	case *m.MiningSetDifficulty:
		p.log.Debugf("new diff: %.0f", msgTyped.GetDifficulty())
	}
	return msg, nil
}

// sourceInterceptor intercepts messages from miner and modifies them if needed, if returns nil then message should be skipped
func (p *Proxy) sourceInterceptor(msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error) {
	switch msgTyped := msg.(type) {
	case *m.MiningConfigure:
		return p.onMiningConfigure(msgTyped)

	case *m.MiningSubscribe:
		return p.onMiningSubscribe(msgTyped)

	case *m.MiningAuthorize:
		return p.onMiningAuthorize(msgTyped)

	case *m.MiningSubmit:
		return p.onMiningSubmit(msgTyped)
	}

	return msg, nil
}

func (p *Proxy) onMiningAuthorize(msgTyped *m.MiningAuthorize) (i.MiningMessageGeneric, error) {
	p.source.SetWorkerName(msgTyped.GetWorkerName())
	p.log = p.log.Named(msgTyped.GetWorkerName())

	msgID := msgTyped.GetID()
	if !minerSubscribeReceived {
		return nil, lib.WrapError(ErrHandshakeSource, fmt.Errorf("MiningAuthorize received before MiningSubscribe"))
	}

	msg := m.NewMiningResultSuccess(msgID)
	err := p.source.Write(context.TODO(), msg)
	if err != nil {
		return nil, lib.WrapError(ErrHandshakeSource, err)
	}

	pwd, ok := p.destURL.User.Password()
	if !ok {
		pwd = ""
	}
	destAuthMsg := m.NewMiningAuthorize(msgID, p.destURL.User.Username(), pwd)

	err = p.dest.Write(context.TODO(), destAuthMsg)
	if err != nil {
		return nil, lib.WrapError(ErrHandshakeDest, err)
	}

	<-p.dest.onceResult(context.Background(), msgID, func(a *m.MiningResult) (i.MiningMessageToPool, error) {
		p.log.Infof("connected to destination: %s", p.destURL.String())
		p.log.Info("handshake completed")
		return a, nil
	})

	return nil, nil
}

func (p *Proxy) onMiningSubscribe(msgTyped *m.MiningSubscribe) (i.MiningMessageGeneric, error) {
	minerSubscribeReceived = true
	msgID := msgTyped.GetID()
	if p.dest == nil {
		destConn, err := p.destFactory(context.TODO(), p.destURL)
		if err != nil {
			return nil, err
		}
		p.dest = destConn
		p.pipe.SetDest(destConn)
		p.pipe.StartDestToSource(context.TODO())
	}

	err := p.dest.Write(context.TODO(), msgTyped)
	if err != nil {
		return nil, lib.WrapError(ErrHandshakeDest, err)
	}

	<-p.dest.onceResult(context.Background(), msgTyped.GetID(), func(a *m.MiningResult) (msg i.MiningMessageToPool, err error) {
		subscribeResult, err := m.ToMiningSubscribeResult(a)
		if err != nil {
			return nil, fmt.Errorf("expected MiningSubscribeResult message, got %s", a.Serialize())
		}

		p.source.SetExtraNonce(subscribeResult.GetExtranonce())
		p.dest.SetExtraNonce(subscribeResult.GetExtranonce())

		subscribeResult.SetID(msgID)

		err = p.source.Write(context.TODO(), subscribeResult)
		if err != nil {
			return nil, lib.WrapError(ErrHandshakeSource, err)
		}
		return nil, nil
	})

	return nil, nil
}

func (p *Proxy) onMiningConfigure(msgTyped *m.MiningConfigure) (i.MiningMessageGeneric, error) {
	msgID := msgTyped.GetID()
	p.source.SetVersionRolling(msgTyped.GetVersionRolling())

	destConn, err := p.destFactory(context.TODO(), p.destURL)
	if err != nil {
		return nil, err
	}

	p.dest = destConn
	p.pipe.SetDest(destConn)
	p.pipe.StartDestToSource(context.TODO())

	err = destConn.Write(context.TODO(), msgTyped)
	if err != nil {
		return nil, lib.WrapError(ErrHandshakeDest, err)
	}

	<-p.dest.onceResult(context.TODO(), msgTyped.GetID(), func(a *m.MiningResult) (msg i.MiningMessageToPool, err error) {
		configureResult, err := m.ToMiningConfigureResult(a)
		if err != nil {
			p.log.Errorf("expected MiningConfigureResult message, got %s", a.Serialize())
			return nil, err
		}

		vr, mask := configureResult.GetVersionRolling(), configureResult.GetVersionRollingMask()
		destConn.SetVersionRolling(vr, mask)
		p.source.SetNegotiatedVersionRollingMask(mask)

		configureResult.SetID(msgID)
		err = p.source.Write(context.TODO(), configureResult)
		if err != nil {
			return nil, lib.WrapError(ErrHandshakeSource, err)
		}

		p.log.Infof("destination connected")
		return nil, nil
	})

	return nil, nil
}

func (p *Proxy) onMiningSubmit(msgTyped *m.MiningSubmit) (i.MiningMessageGeneric, error) {
	p.unansweredMsg.Add(1)

	_, mask := p.dest.GetVersionRolling()
	xn, xn2size := p.dest.GetExtraNonce()

	job, ok := p.dest.GetNotifyMsgJob(msgTyped.GetJobId())
	if !ok {
		p.log.Warnf("cannot find job for this submit (job id %s), skipping", msgTyped.GetJobId())
		// TODO: should we send error to miner?
		err := p.source.Write(context.Background(), m.NewMiningResultJobNotFound(msgTyped.GetID()))
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	diff, ok := Validate(xn, uint(xn2size), uint64(p.dest.GetDiff()), mask, job, msgTyped)
	if !ok {
		p.log.Warnf("validator error: too low difficulty reported by internal validator: expected %.2f actual %d", p.dest.GetDiff(), diff)

		tooLowDiff, _ := m.ParseMiningResult([]byte(`{"id":4,"result":null,"error":[-5,"Too low difficulty",null]}`))
		tooLowDiff.SetID(msgTyped.GetID())
		err := p.source.Write(context.Background(), tooLowDiff)
		if err != nil {
			p.log.Error("cannot write response to miner: ", err)
			return nil, err
		}
		p.unansweredMsg.Done()
		return nil, nil
	}

	p.sourceHR.OnSubmit(p.dest.GetDiff())
	// s.globalHashrate.OnSubmit(s.source.GetWorkerName(), s.dest.GetDiff())

	<-p.dest.onceResult(context.TODO(), msgTyped.GetID(), func(a *m.MiningResult) (i.MiningMessageToPool, error) {
		p.unansweredMsg.Done()
		p.destHR.OnSubmit(p.dest.GetDiff())

		p.onSubmitMutex.RLock()
		if p.onSubmit != nil {
			p.onSubmit.OnSubmit(p.dest.GetDiff())
		}
		p.onSubmitMutex.RUnlock()

		p.log.Debugf("new submit, diff: %d, target: %0.f", diff, p.dest.GetDiff())
		return a, nil
	})

	msgTyped.SetWorkerName(p.destURL.User.Username())

	return msgTyped, nil
}
