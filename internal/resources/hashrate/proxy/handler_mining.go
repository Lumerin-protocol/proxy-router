package proxy

import (
	"context"
	"errors"
	"fmt"

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
	p.proxy.unansweredMsg.Add(1)

	dest := p.proxy.dest
	var res *m.MiningResult

	diff, err := dest.ValidateAndAddShare(msgTyped)
	weAccepted := err == nil

	// if job not found, try searching across all of the connection
	// and replace dest with the one that has the job
	if !weAccepted && errors.Is(err, validator.ErrJobNotFound) {

		d := p.proxy.GetDestByJobID(msgTyped.GetJobId())
		if dest != nil {
			p.log.Warnf("job %s found in different dest %s", msgTyped.GetJobId(), d.GetID())
			diff, err = d.ValidateAndAddShare(msgTyped)
			weAccepted = err == nil
			if weAccepted {
				dest = d
			}
		} else {
			p.log.Warnf("job %s not found", msgTyped.GetJobId())
			res = m.NewMiningResultJobNotFound(msgTyped.GetID())
		}
	}

	if !weAccepted {
		p.proxy.source.GetStats().WeRejectedShares++

		if errors.Is(err, validator.ErrDuplicateShare) {
			p.log.Warnf("duplicate share, jobID %s, msg id: %d", msgTyped.GetJobId(), msgTyped.GetID())
			res = m.NewMiningResultDuplicatedShare(msgTyped.GetID())
		} else if errors.Is(err, validator.ErrLowDifficulty) {
			p.log.Warnf("low difficulty jobID %s, msg id: %d, diff %s", msgTyped.GetJobId(), msgTyped.GetID(), diff)
			res = m.NewMiningResultLowDifficulty(msgTyped.GetID())
		}
	} else {
		p.proxy.source.GetStats().WeAcceptedShares++
		// miner hashrate
		p.proxy.sourceHR.OnSubmit(dest.GetDiff())

		// contract hashrate
		p.proxy.onSubmitMutex.RLock()
		if p.proxy.onSubmit != nil {
			p.proxy.onSubmit(dest.GetDiff())
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
			return
		}

		// send and await submit response from pool
		msgTyped.SetUserName(dest.GetUserName())
		res, err := dest.WriteAwaitRes(ctx, msgTyped)
		if err != nil {
			p.log.Error("cannot write response to pool: ", err)
			p.proxy.cancelRun()
			return
		}
		p.proxy.unansweredMsg.Done()

		if res.(*m.MiningResult).IsError() {
			if weAccepted {
				p.proxy.source.GetStats().WeAcceptedTheyRejected++
				dest.GetStats().WeAcceptedTheyRejected++
				p.log.Warnf("we accepted submit, they rejected with err %s", res.(*m.MiningResult).GetError())
			} else {
				p.log.Warnf("we rejected submit, and they rejected with err %s", res.(*m.MiningResult).GetError())
			}
		} else {
			if weAccepted {
				dest.GetStats().WeAcceptedTheyAccepted++
				p.proxy.destHR.OnSubmit(dest.GetDiff())
				p.log.Infof("new submit, diff: %0.f, hrGHS %d", diff, p.proxy.destHR.GetHashrateGHS())
			} else {
				dest.GetStats().WeRejectedTheyAccepted++
				p.proxy.source.GetStats().WeRejectedTheyAccepted++
				p.log.Warnf("we rejected submit, but dest accepted, diff: %d", diff)
			}
		}
	}(res)

	return nil, nil
}
