package proxy

import (
	"context"
	"errors"
	"fmt"
	"sync"

	gi "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	i "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/interfaces"
	m "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/validator"
)

type HandlerMining struct {
	// deps
	proxy *Proxy
	log   gi.ILogger

	// internal
	unansweredMsg sync.WaitGroup // number of unanswered messages from the source
}

func NewHandlerMining(proxy *Proxy, log gi.ILogger) *HandlerMining {
	return &HandlerMining{
		proxy: proxy,
		log:   log,
	}
}

// sourceInterceptor is called when a message is received from the source after handshake
func (p *HandlerMining) sourceInterceptor(ctx context.Context, msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error) {
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
func (p *HandlerMining) destInterceptor(ctx context.Context, msg i.MiningMessageGeneric) (i.MiningMessageGeneric, error) {
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

// onMiningSubmit is only called when handshake is completed. It doesn't require determinism
// in message ordering, so to improve performance we can use asynchronous pipe
func (p *HandlerMining) onMiningSubmit(ctx context.Context, msgTyped *m.MiningSubmit) (i.MiningMessageGeneric, error) {
	p.unansweredMsg.Add(1)

	diff, err := p.proxy.dest.ValidateAndAddShare(msgTyped)
	weAccepted := err == nil
	var res *m.MiningResult

	if err != nil {
		p.proxy.source.GetStats().WeRejectedShares++

		if errors.Is(err, validator.ErrJobNotFound) {
			res = m.NewMiningResultJobNotFound(msgTyped.GetID())
		} else if errors.Is(err, validator.ErrDuplicateShare) {
			res = m.NewMiningResultDuplicatedShare(msgTyped.GetID())
		} else if errors.Is(err, validator.ErrLowDifficulty) {
			res = m.NewMiningResultLowDifficulty(msgTyped.GetID())
		}

	} else {
		p.proxy.source.GetStats().WeAcceptedShares++
		// miner hashrate
		p.proxy.sourceHR.OnSubmit(p.proxy.dest.GetDiff())

		// contract hashrate
		p.proxy.onSubmitMutex.RLock()
		if p.proxy.onSubmit != nil {
			p.proxy.onSubmit(p.proxy.dest.GetDiff())
		}
		p.proxy.onSubmitMutex.RUnlock()

		res = m.NewMiningResultSuccess(msgTyped.GetID())
	}

	// workername hashrate
	// s.globalHashrate.OnSubmit(s.source.GetWorkerName(), s.dest.GetDiff())

	// does not wait for response from destination pool
	// TODO: implement buffering for source/dest messages
	// to avoid blocking source/dest when one of them is slow
	// and fix error handling to avoid p.cancelRun
	go func(res1 *m.MiningResult) {
		err = p.proxy.source.Write(ctx, res1)
		if err != nil {
			p.log.Error("cannot write response to miner: ", err)
			p.proxy.cancelRun()
		}

		// send and await submit response from pool
		msgTyped.SetWorkerName(p.proxy.dest.GetWorkerName())
		res, err := p.proxy.dest.WriteAwaitRes(ctx, msgTyped)
		if err != nil {
			p.log.Error("cannot write response to pool: ", err)
			p.proxy.cancelRun()
		}
		p.unansweredMsg.Done()

		if res.(*m.MiningResult).IsError() {
			if weAccepted {
				p.proxy.source.GetStats().WeAcceptedTheyRejected++
				p.proxy.dest.GetStats().WeAcceptedTheyRejected++
			}
		} else {
			if weAccepted {
				p.proxy.dest.GetStats().WeAcceptedTheyAccepted++
				p.proxy.destHR.OnSubmit(p.proxy.dest.GetDiff())
				p.log.Debugf("new submit, diff: %0.f, target: %0.f", diff, p.proxy.dest.GetDiff())
			} else {
				p.proxy.dest.GetStats().WeRejectedTheyAccepted++
				p.proxy.source.GetStats().WeRejectedTheyAccepted++
				p.log.Warnf("we rejected submit, but dest accepted, diff: %d, target: %0.f", diff, p.proxy.dest.GetDiff())
			}
		}
	}(res)

	return nil, nil
}
