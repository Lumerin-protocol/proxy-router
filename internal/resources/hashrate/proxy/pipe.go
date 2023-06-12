package proxy

import (
	"context"
	"fmt"

	gi "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
)

type Pipe struct {
	// state
	reconnectCh             chan struct{} // signal to reconnect, when dest changed
	destToSourceStartSignal chan struct{} // signal to start reading from destination
	poolMinerCancel         context.CancelFunc
	minerPoolCancel         context.CancelFunc
	sourceToDestTask        *lib.Task
	destToSourceTask        *lib.Task

	// deps
	source            StratumReadWriter // initiator of the communication, miner
	dest              StratumReadWriter // receiver of the communication, pool
	sourceInterceptor Interceptor
	destInterceptor   Interceptor

	log gi.ILogger
}

// NewPipe creates a new pipe between source and dest, allowing to intercept messages and separately control
// start and stop on both directions of the duplex
func NewPipe(source, dest StratumReadWriter, sourceInterceptor, destInterceptor Interceptor, log gi.ILogger) *Pipe {
	pipe := &Pipe{
		source:            source,
		dest:              dest,
		sourceInterceptor: sourceInterceptor,
		destInterceptor:   destInterceptor,
		log:               log,
	}

	sourceToDestTask := lib.NewTaskFunc(pipe.sourceToDest)
	destToSourceTask := lib.NewTaskFunc(pipe.destToSource)

	pipe.sourceToDestTask = sourceToDestTask
	pipe.destToSourceTask = destToSourceTask

	return pipe
}

func (p *Pipe) Run(ctx context.Context) error {
	var err error

	select {
	case <-p.sourceToDestTask.Done():
		err = p.sourceToDestTask.Err()
		<-p.destToSourceTask.Stop()
	case <-p.destToSourceTask.Done():
		err = p.destToSourceTask.Err()
		<-p.sourceToDestTask.Stop()
	}

	return err
}

func (p *Pipe) destToSource(ctx context.Context) error {
	err := pipe(ctx, p.dest, p.source, p.destInterceptor)
	if err != nil {
		return fmt.Errorf("dest to source pipe err: %w", err)
	}
	return nil
}

func (p *Pipe) sourceToDest(ctx context.Context) error {
	err := pipe(ctx, p.source, p.dest, p.sourceInterceptor)
	if err != nil {
		return fmt.Errorf("source to dest pipe err: %w", err)
	}
	return nil
}

func pipe(ctx context.Context, from StratumReadWriter, to StratumReadWriter, interceptor Interceptor) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := from.Read(ctx)
		if err != nil {
			return fmt.Errorf("pool read err: %w", err)
		}

		msg, err = interceptor(msg)
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

		err = to.Write(ctx, msg)
		if err != nil {
			return fmt.Errorf("miner write err: %w", err)
		}
	}
}

func (p *Pipe) SetDest(dest StratumReadWriter) {
	p.dest = dest
}

func (p *Pipe) StartSourceToDest(ctx context.Context) {
	p.sourceToDestTask.Start(ctx)
}
func (p *Pipe) StartDestToSource(ctx context.Context) {
	p.destToSourceTask.Start(ctx)
}
func (p *Pipe) StopSourceToDest() <-chan struct{} {
	return p.sourceToDestTask.Stop()
}
func (p *Pipe) StopDestToSource() <-chan struct{} {
	return p.destToSourceTask.Stop()
}
