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
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
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

type ResultHandler = func(ctx context.Context, a *stratumv1_message.MiningResult) (msg i.MiningMessageToPool, err error)

type Proxy struct {
	// config
	ID             string
	destWorkerName string
	submitErrLimit int
	onFault        func(context.Context) // called when proxy becomes faulty (e.g. when submit error limit is reached
	destURL        *url.URL              // destination URL

	// state
	reconnectCh             chan struct{} // signal to reconnect, when dest changed
	poolMinerCancel         context.CancelFunc
	minerPoolCancel         context.CancelFunc
	unansweredMsg           sync.WaitGroup     // number of unanswered messages from the source
	destToSourceStartSignal chan struct{}      // signal to start reading from destination
	sourceHR                *hashrate.Hashrate // hashrate of the source validated by the proxy
	destHR                  *hashrate.Hashrate // hashrate of the destination validated by the destination

	// deps
	source         *SourceConn           // initiator of the communication, miner
	dest           *DestConn             // receiver of the communication, pool
	destMap        map[string]*DestConn  // map of all available destinations (pools) currently connected to the single source (miner)
	onSubmit       HashrateCounter       // callback to update contract hashrate
	onSubmitMutex  sync.RWMutex          // mutex to protect onSubmit
	globalHashrate GlobalHashrateCounter // callback to update global hashrate per worker
	destFactory    DestConnFactory       // factory to create new destination connections
	log            gi.ILogger
}

// TODO: pass connection factory for destURL
func NewProxy(ID string, source *SourceConn, destFactory DestConnFactory, destURL *url.URL, log gi.ILogger) *Proxy {
	destMap := make(map[string]*DestConn)

	return &Proxy{
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
}

var (
	firstMessageReadTimeout  = 5 * time.Second
	firstMessageWriteTimeout = 5 * time.Second

	minerSubscribeReceived = false
	// minerConfigureReceived = false
	// minerAuthorizeReceived = false
	// pooConfigureSent       = false

)

func (p *Proxy) Run(ctx context.Context) error {
	parentCtx, parentCancel := context.WithCancel(ctx)
	defer parentCancel()

	// reroute loop
	for {
		p.reconnectCh = make(chan struct{})
		var (
			sourceToDestErrCh = make(chan error)
			destToSourceErrCh = make(chan error)
		)

		minerPoolCtx, minerPoolCancel := context.WithCancel(parentCtx)
		p.minerPoolCancel = minerPoolCancel

		go func() {
			err := p.sourceToDest(minerPoolCtx)
			if err != nil {
				sourceToDestErrCh <- err
				close(sourceToDestErrCh)
			}
			p.log.Warn("miner to pool done")
		}()

		poolMinerCtx, poolMinerCancel := context.WithCancel(parentCtx)
		p.poolMinerCancel = poolMinerCancel

		go func() {
			<-p.destToSourceStartSignal // wait for the signal to start reading from destination

			// enables autorun in the background
			go func() {
				err := p.dest.Run(ctx)
				if err != nil {
					p.log.Error("destination run error", err)
				}
				err = p.dest.conn.Close()
				if err != nil {
					p.log.Error("error closing destination connection", err)
				}
			}()

			err := p.destToSource(poolMinerCtx)
			if err != nil {
				destToSourceErrCh <- err
				close(destToSourceErrCh)
			}
			p.log.Warn("pool to miner done")
		}()

		var err error
		// waiting for one of the routines to return
		select {
		case err = <-sourceToDestErrCh:
		case err = <-destToSourceErrCh:
		}

		// if parent ctx is cancelled then exit run
		if parentCtx.Err() != nil {
			<-sourceToDestErrCh
			<-destToSourceErrCh
			return lib.WrapError(ErrProxy, parentCtx.Err())
		}

		// otherwise resume passthrough with different destination
		p.log.Debugf("proxy error: %v", err)

		p.log.Debugf("waiting for both routines to stop")
		minerPoolCancel()
		poolMinerCancel()

		<-sourceToDestErrCh
		<-destToSourceErrCh

		if errors.Is(err, context.Canceled) {
			p.log.Debugf("waiting for reconnect signal")
			<-p.reconnectCh // wait for reconnect signal
			p.log.Debugf("got reconnect signal")
		} else {
			p.log.Warnf("waiting for reconnect signal")
			return lib.WrapError(ErrProxy, err)
		}
	}
}

func (p *Proxy) ChangeDest(ctx context.Context, newDestURL *url.URL) error {
	p.log.Warnf("changing destination to %s", newDestURL.String())

	newDest, err := p.destFactory(ctx, newDestURL)
	if err != nil {
		return lib.WrapError(ErrConnectDest, err)
	}

	var (
		msgID        = 1
		awaitMsgSent chan struct{}
		runErrCh     = make(chan error)
	)

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()
	go func() {
		p.log.Warnf("running new destination")

		err := newDest.Run(runCtx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				p.log.Debugf("running new dest cancelled")
			} else {
				p.log.Errorf("running new dest error: %v", err)
			}
		}

		runErrCh <- err
	}()

	newDest.AutoReadStart()
	p.log.Debugf("new dest autoread started")

	// 1. MINING.CONFIGURE
	// if miner has version mask enabled, send it to the pool
	if p.source.GetNegotiatedVersionRollingMask() != "" {
		// using the same version mask as the miner negotiated during the prev connection
		cfgMsg := stratumv1_message.NewMiningConfigure(msgID, nil)
		_, minBits := p.source.GetVersionRolling()
		cfgMsg.SetVersionRolling(p.source.GetNegotiatedVersionRollingMask(), minBits)

		writeCtx, cancel := context.WithTimeout(context.Background(), firstMessageWriteTimeout)
		defer cancel()
		err := newDest.Write(writeCtx, stratumv1_message.NewMiningConfigure(msgID, nil))
		if err != nil {
			return lib.WrapError(ErrConnectDest, err)
		}
		p.log.Debugf("configure sent")

		awaitMsgSent = make(chan struct{})
		newDest.onceResult(ctx, msgID, func(ctx context.Context, msg *stratumv1_message.MiningResult) (i.MiningMessageToPool, error) {
			cfgRes, err := stratumv1_message.ToMiningConfigureResult(msg)
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
			close(awaitMsgSent)
			return nil, nil
		})
		<-awaitMsgSent
		p.log.Debugf("configure result received")
	}

	// 2. MINING.SUBSCRIBE
	msgID++
	writeCtx, _ := context.WithTimeout(context.Background(), firstMessageWriteTimeout)
	err = newDest.Write(writeCtx, stratumv1_message.NewMiningSubscribe(msgID, "stratum-proxy", "1.0.0"))
	if err != nil {
		return lib.WrapError(ErrConnectDest, err)
	}
	p.log.Debugf("subscribe sent")

	awaitMsgSent = make(chan struct{})
	newDest.onceResult(ctx, msgID, func(ctx context.Context, msg *stratumv1_message.MiningResult) (i.MiningMessageToPool, error) {
		subRes, err := stratumv1_message.ToMiningSubscribeResult(msg)
		if err != nil {
			return nil, err
		}
		if subRes.IsError() {
			return nil, fmt.Errorf("pool returned error: %s", subRes.GetError())
		}

		newDest.SetExtraNonce(subRes.GetExtranonce())

		close(awaitMsgSent)
		return nil, nil
	})
	<-awaitMsgSent
	p.log.Warnf("subscribe result received")

	// 3. MINING.AUTHORIZE
	msgID++
	writeCtx, _ = context.WithTimeout(context.Background(), firstMessageWriteTimeout)
	user := newDestURL.User.Username()
	pwd, _ := newDestURL.User.Password()
	err = newDest.Write(writeCtx, stratumv1_message.NewMiningAuthorize(msgID, user, pwd))
	if err != nil {
		return lib.WrapError(ErrConnectDest, err)
	}
	p.log.Debugf("authorize sent")

	awaitMsgSent = make(chan struct{})
	newDest.onceResult(ctx, msgID, func(ctx context.Context, msg *stratumv1_message.MiningResult) (i.MiningMessageToPool, error) {
		if msg.IsError() {
			return nil, lib.WrapError(ErrConnectDest, lib.WrapError(ErrNotAuthorizedPool, fmt.Errorf("%s", msg.GetError())))
		}
		close(awaitMsgSent)
		return nil, nil
	})
	<-awaitMsgSent
	p.log.Debugf("authorize success")

	// HANDSHAKE DONE

	// stop running new dest
	runCancel()
	<-runErrCh
	p.log.Debugf("stopped new dest")

	// stop source and old dest
	p.minerPoolCancel()
	p.poolMinerCancel()
	p.log.Warnf("stopped source and old dest")

	// TODO: wait to stop?

	// set old dest to autoread mode
	p.dest.AutoReadStart()
	p.log.Warnf("set old dest to autoread")

	// resend relevant notifications to the miner
	// 1. SET_EXTRANONCE
	err = p.source.Write(ctx, stratumv1_message.NewMiningSetExtranonce(newDest.GetExtraNonce()))
	if err != nil {
		return lib.WrapError(ErrChangeDest, err)
	}
	p.source.SetExtraNonce(newDest.GetExtraNonce())
	p.log.Warnf("extranonce sent")

	// 2. SET_DIFFICULTY
	err = p.source.Write(ctx, stratumv1_message.NewMiningSetDifficulty(p.dest.GetDiff()))
	if err != nil {
		return lib.WrapError(ErrChangeDest, err)
	}
	p.log.Warnf("set difficulty sent")

	// TODO: 3. SET_VERSION_MASK

	// 4. NOTIFY
	msg, ok := p.dest.notifyMsgs.At(-1)
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

	p.dest = newDest
	p.destURL = newDestURL
	p.log.Infof("changing dest success")

	// resume reconnection loop
	close(p.reconnectCh)
	p.log.Debugf("resume reconnect loop")

	return nil
}

func (p *Proxy) destToSource(ctx context.Context) error {
	for {
		msg, err := p.dest.Read(ctx)
		if err != nil {
			return fmt.Errorf("pool read err: %w", err)
		}

		msg, err = p.destInterceptor(msg)
		if err != nil {
			return fmt.Errorf("dest interceptor err: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if msg == nil {
			continue
		}

		err = p.source.Write(ctx, msg)
		if err != nil {
			return fmt.Errorf("miner write err: %w", err)
		}
	}
}

func (p *Proxy) sourceToDest(ctx context.Context) error {
	for {
		msg, err := p.source.Read(ctx)
		if err != nil {
			return fmt.Errorf("miner read err: %w", err)
		}

		msg, err = p.sourceInterceptor(msg)
		if err != nil {
			return fmt.Errorf("source interceptor err: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if msg == nil {
			continue
		}

		err = p.dest.Write(ctx, msg)
		if err != nil {
			return fmt.Errorf("pool write err: %w", err)
		}
	}
}

func (p *Proxy) destInterceptor(msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error) {
	switch msgTyped := msg.(type) {
	case *stratumv1_message.MiningSetDifficulty:
		p.log.Debugf("new diff: %.0f", msgTyped.GetDifficulty())
	}
	return msg, nil
}

// sourceInterceptor intercepts messages from miner and modifies them if needed, if returns nil then message should be skipped
func (p *Proxy) sourceInterceptor(msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error) {
	switch msgTyped := msg.(type) {
	case *stratumv1_message.MiningConfigure:
		// TODO: throw error if handshake is already done
		// minerConfigureReceived = true
		msgID := msgTyped.GetID()
		p.source.SetVersionRolling(msgTyped.GetVersionRolling())

		destConn, err := p.destFactory(context.TODO(), p.destURL)
		if err != nil {
			return nil, err
		}

		p.dest = destConn
		close(p.destToSourceStartSignal) // close the signal to start reading from destination

		writeCtx, _ := context.WithTimeout(context.Background(), firstMessageWriteTimeout)
		err = destConn.Write(writeCtx, msgTyped)
		if err != nil {
			return nil, lib.WrapError(ErrHandshakeDest, err)
		}

		p.dest.onceResult(context.TODO(), msgTyped.GetID(), func(ctx context.Context, a *stratumv1_message.MiningResult) (msg i.MiningMessageToPool, err error) {
			configureResult, err := stratumv1_message.ToMiningConfigureResult(a)
			if err != nil {
				p.log.Errorf("expected MiningConfigureResult message, got %s", a.Serialize())
				return nil, err
			}

			vr, mask := configureResult.GetVersionRolling(), configureResult.GetVersionRollingMask()
			destConn.SetVersionRolling(vr, mask)
			p.source.SetNegotiatedVersionRollingMask(mask)

			configureResult.SetID(msgID)
			writeCtx, _ = context.WithTimeout(context.Background(), firstMessageWriteTimeout)
			err = p.source.Write(writeCtx, configureResult)
			if err != nil {
				return nil, lib.WrapError(ErrHandshakeSource, err)
			}

			p.log.Infof("destination connected")
			return nil, nil
		})

		return nil, nil

	case *stratumv1_message.MiningSubscribe:
		minerSubscribeReceived = true
		msgID := msgTyped.GetID()
		if p.dest == nil {
			destConn, err := p.destFactory(context.TODO(), p.destURL)
			if err != nil {
				return nil, err
			}
			p.dest = destConn
			close(p.destToSourceStartSignal) // close the signal to start reading from destination
		}

		writeCtx, _ := context.WithTimeout(context.Background(), firstMessageWriteTimeout)
		err := p.dest.Write(writeCtx, msgTyped)
		if err != nil {
			return nil, lib.WrapError(ErrHandshakeDest, err)
		}

		// read the response from the destination
		p.dest.onceResult(context.Background(), msgTyped.GetID(), func(ctx context.Context, a *stratumv1_message.MiningResult) (msg i.MiningMessageToPool, err error) {
			subscribeResult, err := stratumv1_message.ToMiningSubscribeResult(a)
			if err != nil {
				return nil, fmt.Errorf("expected MiningSubscribeResult message, got %s", a.Serialize())
			}

			p.source.SetExtraNonce(subscribeResult.GetExtranonce())
			p.dest.SetExtraNonce(subscribeResult.GetExtranonce())

			subscribeResult.SetID(msgID)

			writeCtx, cancel := context.WithTimeout(context.Background(), firstMessageWriteTimeout)
			err = p.source.Write(writeCtx, subscribeResult)
			if err != nil {
				cancel()
				return nil, lib.WrapError(ErrHandshakeSource, err)
			}
			cancel()
			return nil, nil
		})

		return nil, nil

	case *stratumv1_message.MiningAuthorize:
		// minerAuthorizeReceived = true
		p.source.SetWorkerName(msgTyped.GetWorkerName())
		p.log = p.log.Named(msgTyped.GetWorkerName())

		msgID := msgTyped.GetID()
		if !minerSubscribeReceived {
			return nil, lib.WrapError(ErrHandshakeSource, fmt.Errorf("MiningAuthorize received before MiningSubscribe"))
		}

		// reply immediately to the source
		msg := stratumv1_message.NewMiningResultSuccess(msgID)
		writeCtx, _ := context.WithTimeout(context.Background(), firstMessageWriteTimeout)
		err := p.source.Write(writeCtx, msg)
		if err != nil {
			return nil, lib.WrapError(ErrHandshakeSource, err)
		}

		// connect to destination
		pwd, ok := p.destURL.User.Password()
		if !ok {
			pwd = ""
		}
		destAuthMsg := stratumv1_message.NewMiningAuthorize(msgID, p.destURL.User.Username(), pwd)

		writeCtx, _ = context.WithTimeout(context.Background(), firstMessageReadTimeout)
		err = p.dest.Write(writeCtx, destAuthMsg)
		if err != nil {
			return nil, lib.WrapError(ErrHandshakeDest, err)
		}

		p.dest.onceResult(context.Background(), msgID, func(ctx context.Context, a *stratumv1_message.MiningResult) (i.MiningMessageToPool, error) {
			p.log.Infof("connected to destination: %s", p.destURL.String())
			p.log.Info("handshake completed")
			return a, nil
		})

		return nil, nil

	case *stratumv1_message.MiningSubmit:
		p.unansweredMsg.Add(1)

		_, mask := p.dest.GetVersionRolling()
		xn, xn2size := p.dest.GetExtraNonce()

		job, ok := p.dest.GetNotifyMsgJob(msgTyped.GetJobId())
		if !ok {
			p.log.Warnf("cannot find job for this submit (job id %s), skipping", msgTyped.GetJobId())
			// TODO: should we send error to miner?
			err := p.source.Write(context.Background(), stratumv1_message.NewMiningResultJobNotFound(msgTyped.GetID()))
			if err != nil {
				return nil, err
			}
			return nil, nil
		}

		diff, ok := Validate(xn, uint(xn2size), uint64(p.dest.GetDiff()), mask, job, msgTyped)
		if !ok {
			p.log.Warnf("validator error: too low difficulty reported by internal validator: expected %.2f actual %d", p.dest.GetDiff(), diff)

			tooLowDiff, _ := stratumv1_message.ParseMiningResult([]byte(`{"id":4,"result":null,"error":[-5,"Too low difficulty",null]}`))
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

		p.dest.onceResult(context.TODO(), msgTyped.GetID(), func(ctx context.Context, a *stratumv1_message.MiningResult) (i.MiningMessageToPool, error) {
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

	return msg, nil
}
