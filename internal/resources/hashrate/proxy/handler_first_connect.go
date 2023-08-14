package proxy

import (
	"context"
	"errors"
	"fmt"

	gi "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	i "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/interfaces"
	m "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
	sm "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
)

type HandlerFirstConnect struct {
	proxy *Proxy

	handshakePipe       *pipeSync
	handshakePipeTsk    *lib.Task
	cancelHandshakePipe context.CancelFunc

	log gi.ILogger
}

func NewHandlerFirstConnect(proxy *Proxy, log gi.ILogger) *HandlerFirstConnect {
	return &HandlerFirstConnect{
		proxy: proxy,
		log:   log,
	}
}

func (p *HandlerFirstConnect) Connect(ctx context.Context) error {
	p.handshakePipe = NewPipeSync(p.proxy.source, p.proxy.dest, p.handleSource, p.handleDest)
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

func (p *HandlerFirstConnect) handleSource(ctx context.Context, msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error) {
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
		return nil, p.proxy.dest.Write(context.Background(), msgTyped)
	}
}

func (p *HandlerFirstConnect) handleDest(ctx context.Context, msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error) {
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
		return nil, p.proxy.source.Write(ctx, msgOut)
	}

	return nil, nil
}

// The following handlers are responsible for managing the initial connection of the miner to the proxy and destination.
// This step requires special handling due to the "coupled" interaction between parties, unlike the destination change process,
// where the pool connection is established first, and then the miner is switched to it. This "coupled" interaction is intentionally
// designed to enable the negotiation of the version rolling mask. It's important to note that all of these handlers require
// performing reads and writes within the same goroutine. Additionally, other response handlers (identified by message ID) must be
// called right after receiving the message. This ensures that the order of messages is deterministic. If the order of messages
// during the handshake is not enforced, there is a possibility that miners may fail, for example, if the "set_version_mask"
// message is sent to the miner before receiving the "configure" result.

func (p *HandlerFirstConnect) onMiningConfigure(ctx context.Context, msgTyped *m.MiningConfigure) error {
	p.proxy.source.SetVersionRolling(msgTyped.GetVersionRolling())

	destConn, err := p.proxy.destFactory(ctx, p.proxy.destURL)
	if err != nil {
		return err
	}

	p.proxy.dest = destConn
	p.handshakePipe.SetStream2(destConn)
	p.handshakePipe.StartStream2()

	err = p.proxy.dest.Write(ctx, msgTyped)
	if err != nil {
		return lib.WrapError(ErrHandshakeDest, err)
	}

	p.proxy.dest.onceResult(ctx, msgTyped.GetID(), func(res *sm.MiningResult) (msg i.MiningMessageWithID, err error) {
		configureResult, err := m.ToMiningConfigureResult(res)
		if err != nil {
			p.log.Errorf("expected MiningConfigureResult message, got %s", res.Serialize())
			return nil, err
		}

		vr, mask := configureResult.GetVersionRolling(), configureResult.GetVersionRollingMask()
		destConn.SetVersionRolling(vr, mask)
		p.proxy.source.SetNegotiatedVersionRollingMask(mask)

		configureResult.SetID(msgTyped.GetID())
		err = p.proxy.source.Write(ctx, configureResult)
		if err != nil {
			return nil, lib.WrapError(ErrHandshakeSource, err)
		}

		err = p.proxy.source.Write(ctx, m.NewMiningSetVersionMask(configureResult.GetVersionRollingMask()))
		if err != nil {
			return nil, lib.WrapError(ErrHandshakeSource, err)
		}
		return nil, nil
	})

	return nil
}

func (p *HandlerFirstConnect) onMiningSubscribe(ctx context.Context, msgTyped *m.MiningSubscribe) error {
	minerSubscribeReceived = true

	if p.proxy.dest == nil {
		destConn, err := p.proxy.destFactory(ctx, p.proxy.destURL)
		if err != nil {
			return err
		}

		p.proxy.dest = destConn
		p.handshakePipe.SetStream2(destConn)
		p.handshakePipe.StartStream2()
	}

	err := p.proxy.dest.Write(ctx, msgTyped)
	if err != nil {
		return lib.WrapError(ErrHandshakeDest, err)
	}

	p.proxy.dest.onceResult(ctx, msgTyped.GetID(), func(res *sm.MiningResult) (msg i.MiningMessageWithID, err error) {
		subscribeResult, err := m.ToMiningSubscribeResult(res)
		if err != nil {
			return nil, fmt.Errorf("expected MiningSubscribeResult message, got %s", res.Serialize())
		}

		p.proxy.source.SetExtraNonce(subscribeResult.GetExtranonce())
		p.proxy.dest.SetExtraNonce(subscribeResult.GetExtranonce())

		subscribeResult.SetID(msgTyped.GetID())

		err = p.proxy.source.Write(ctx, subscribeResult)
		if err != nil {
			return nil, lib.WrapError(ErrHandshakeSource, err)
		}
		return nil, nil
	})

	return nil
}

func (p *HandlerFirstConnect) onMiningAuthorize(ctx context.Context, msgTyped *m.MiningAuthorize) error {
	p.proxy.source.SetWorkerName(msgTyped.GetWorkerName())
	p.log = p.log.Named(msgTyped.GetWorkerName())

	msgID := msgTyped.GetID()
	if !minerSubscribeReceived {
		return lib.WrapError(ErrHandshakeSource, fmt.Errorf("MiningAuthorize received before MiningSubscribe"))
	}

	msg := m.NewMiningResultSuccess(msgID)
	err := p.proxy.source.Write(ctx, msg)
	if err != nil {
		return lib.WrapError(ErrHandshakeSource, err)
	}

	_, workerName, _ := lib.SplitUsername(msgTyped.GetWorkerName())
	lib.SetWorkerName(p.proxy.destURL, workerName)
	userName := p.proxy.destURL.User.Username()

	p.proxy.dest.SetWorkerName(userName)

	pwd, ok := p.proxy.destURL.User.Password()
	if !ok {
		pwd = ""
	}
	destAuthMsg := m.NewMiningAuthorize(msgID, userName, pwd)

	err = p.proxy.dest.Write(ctx, destAuthMsg)
	if err != nil {
		return lib.WrapError(ErrHandshakeDest, err)
	}

	p.proxy.dest.onceResult(ctx, msgID, func(res *sm.MiningResult) (msg i.MiningMessageWithID, err error) {
		if res.IsError() {
			return nil, lib.WrapError(ErrHandshakeDest, fmt.Errorf("cannot authorize in dest pool: %s", res.GetError()))
		}
		p.log.Infof("connected to destination: %s", p.proxy.destURL.String())
		p.log.Info("handshake completed")

		p.proxy.destMap.Store(p.proxy.dest)

		// close
		p.cancelHandshakePipe()

		return nil, nil
	})

	return nil
}
