package protocol

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

const (
	MAX_PAUSE_DURATION             = 15 * time.Second
	DEFAULT_SUBMIT_ERR_COUNT_LIMIT = 15
)

type stratumV1MinerModel struct {
	poolConn  StratumV1DestConn
	minerConn StratumV1SourceConn
	validator *hashrate.Hashrate

	difficulty    int64
	onSubmit      interfaces.IHashrate
	onSubmitMutex sync.RWMutex // guards onSubmit

	globalHashrate interfaces.GlobalHashrate

	configureMsgReq *stratumv1_message.MiningConfigure

	unansweredMsg sync.WaitGroup // inverted semaphore that counts messages that were unanswered

	poolMinerCancel context.CancelFunc
	minerPoolCancel context.CancelFunc

	reconnectCh chan struct{}

	submitErrCount atomic.Int32 // counter of successive submit errors, it is the case when set_extranonce is not supported
	submitErrLimit int          // after this amount of errors miner will be considered faulty
	isFaulty       bool         // true when number of successive errors exceeded limit, miner will be exculded from fulfilling contracts
	onFault        func(ctx context.Context)

	workerName string

	log interfaces.ILogger
}

func NewStratumV1MinerModel(poolPool StratumV1DestConn, minerConn StratumV1SourceConn, validator *hashrate.Hashrate, submitErrLimit int, globalHashrate interfaces.GlobalHashrate, log interfaces.ILogger) *stratumV1MinerModel {
	return &stratumV1MinerModel{
		poolConn:       poolPool,
		minerConn:      minerConn,
		validator:      validator,
		unansweredMsg:  sync.WaitGroup{},
		submitErrCount: atomic.Int32{},
		submitErrLimit: submitErrLimit,
		isFaulty:       false,
		globalHashrate: globalHashrate,

		log: log,
	}
}

func (s *stratumV1MinerModel) Connect() error {
	for {
		m, err := s.minerConn.Read(context.TODO())
		if err != nil {
			s.log.Error(err)
			return err
		}

		switch typedMessage := m.(type) {
		case *stratumv1_message.MiningConfigure:
			id := typedMessage.GetID()

			s.configureMsgReq = typedMessage
			msg, err := s.poolConn.SendPoolRequestWait(typedMessage)
			if err != nil {
				return err
			}
			confRes, err := stratumv1_message.ToMiningConfigureResult(msg.Copy())
			if err != nil {
				return err
			}

			confRes.SetID(id)
			err = s.minerConn.Write(context.TODO(), confRes)
			if err != nil {
				return err
			}

		case *stratumv1_message.MiningSubscribe:
			extranonce, size := s.poolConn.GetExtranonce()
			msg := stratumv1_message.NewMiningSubscribeResult(extranonce, size)

			msg.SetID(typedMessage.GetID())
			err := s.minerConn.Write(context.TODO(), msg)
			if err != nil {
				return err
			}

		case *stratumv1_message.MiningAuthorize:
			s.setWorkerName(typedMessage.GetWorkerName())

			msg, _ := stratumv1_message.ParseMiningResult([]byte(`{"id":47,"result":true,"error":null}`))
			msg.SetID(typedMessage.GetID())
			err := s.minerConn.Write(context.TODO(), msg)
			if err != nil {
				return err
			}
			// auth successful
			return nil
		}
	}
}

func (s *stratumV1MinerModel) Run(ctx context.Context) error {
	defer s.Cleanup()

	err := s.Connect()
	if err != nil {
		s.log.Error(err)
		return err
	}

	subCtx, cancel := context.WithCancel(ctx)
	errCh := make(chan error, 2)

	for { // reconnection to different destinations loop
		s.reconnectCh = make(chan struct{})
		if err := s.poolConn.ResendRelevantNotifications(ctx); err != nil {
			s.log.Errorf("error during resending relevant notifications %s", err)
		}

		poolMinerCtx, poolMinerCancel := context.WithCancel(subCtx)
		s.poolMinerCancel = poolMinerCancel

		go func() {
			err := s.poolToMiner(poolMinerCtx)
			if err != nil {
				errCh <- err
			}
			s.log.Warn("pool 2 miner done")
		}()

		minerPoolCtx, minerPoolCancel := context.WithCancel(subCtx)
		s.minerPoolCancel = minerPoolCancel

		go func() {
			err := s.minerToPool(minerPoolCtx)
			if err != nil {
				errCh <- err
			}
			s.log.Warn("miner 2 pool done")
		}()

		err = <-errCh

		// if parent context cancelled then break outer loop
		if subCtx.Err() != nil {
			s.log.Debugf("outer context cancelled, exiting Run method")
			cancel()
			<-errCh // wait for the second routine
			err = subCtx.Err()
			break
		}

		// if inner context cancelled just reconnect to next destination
		if errors.Is(err, context.Canceled) {
			s.log.Debugf("inner context cancelled, reconnection")

			s.log.Debugf("waiting for second routine")
			<-errCh // wait for the second routine

			s.log.Debugf("waiting for reconnect signal")
			<-s.reconnectCh // wait for reconnect signal
			continue
		}

		// if any other connection error break outer loop
		cancel()
		err = fmt.Errorf("miner model error: %w", err)
		<-errCh // wait for the second routine
		break
	}

	return err
}

func (s *stratumV1MinerModel) poolToMiner(ctx context.Context) error {
	for {
		msg, err := s.poolConn.Read(ctx)
		if err != nil {
			return fmt.Errorf("pool read err: %w", err)
		}

		s.poolInterceptor(msg)

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err = s.minerConn.Write(ctx, msg)
		if err != nil {
			return fmt.Errorf("miner write err: %w", err)
		}
	}
}

func (s *stratumV1MinerModel) minerToPool(ctx context.Context) error {
	for {
		msg, err := s.minerConn.Read(ctx)
		if err != nil {
			return fmt.Errorf("miner read err: %w", err)
		}

		s.minerInterceptor(msg)

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err = s.poolConn.Write(ctx, msg)
		if err != nil {
			return fmt.Errorf("pool write err: %w", err)
		}
	}
}

func (s *stratumV1MinerModel) OnFault(cb func(ctx context.Context)) {
	s.onFault = cb
}

func (s *stratumV1MinerModel) IsFaulty() bool {
	return s.isFaulty
}

func (s *stratumV1MinerModel) minerInterceptor(msg stratumv1_message.MiningMessageGeneric) {
	switch typed := msg.(type) {
	case *stratumv1_message.MiningSubmit:
		s.unansweredMsg.Add(1)

		s.poolConn.RegisterResultHandler(typed.GetID(), func(a stratumv1_message.MiningResult) stratumv1_message.MiningMessageGeneric {
			s.unansweredMsg.Done()

			if a.IsError() {
				s.log.Warnf("error during submit: %s msg ID %d", a.GetError(), a.ID)
				if s.submitErrLimit == 0 {
					return &a
				}
				errCount := s.submitErrCount.Add(1)
				if errCount > int32(s.submitErrLimit) {
					s.log.Warnf("consecutive submit error count(%d) exceded limit(%d)", errCount, s.submitErrLimit)
					s.isFaulty = true
					if s.onFault != nil {
						s.onFault(context.Background())
					}
					s.submitErrCount.Store(0)
				}
				return &a
			}
			s.submitErrCount.Store(0)
			s.validator.OnSubmit(s.difficulty)
			s.globalHashrate.OnSubmit(s.workerName, s.difficulty)

			s.onSubmitMutex.RLock()
			defer s.onSubmitMutex.RUnlock()
			if s.onSubmit != nil {
				s.onSubmit.OnSubmit(s.difficulty)
			}

			return &a
		})

	}
}

func (s *stratumV1MinerModel) poolInterceptor(msg stratumv1_message.MiningMessageGeneric) {
	switch m := msg.(type) {
	case *stratumv1_message.MiningSetDifficulty:
		// TODO: some pools return difficulty in float, decide if we need that kind of precision
		s.difficulty = int64(m.GetDifficulty())
	}
}

func (s *stratumV1MinerModel) ChangeDest(ctx context.Context, dest interfaces.IDestination, onSubmit interfaces.IHashrate) error {
	s.minerPoolCancel()

	s.log.Debug("waiting for responses")
	s.unansweredMsg.Wait()
	s.log.Debug("waiting for responses finished")

	s.poolMinerCancel()

	err := s.poolConn.SetDest(ctx, dest, s.configureMsgReq)
	if err != nil {
		return err
	}
	s.setOnSubmit(onSubmit)
	s.reconnectCh <- struct{}{}

	return nil
}

func (s *stratumV1MinerModel) GetDest() interfaces.IDestination {
	return s.poolConn.GetDest()
}

func (s *stratumV1MinerModel) GetID() string {
	return s.minerConn.GetID()
}

func (s *stratumV1MinerModel) GetHashRateGHS() int {
	return s.validator.GetHashrateGHS()
}

func (s *stratumV1MinerModel) GetHashRate() interfaces.Hashrate {
	return s.validator
}

func (s *stratumV1MinerModel) GetCurrentDifficulty() int {
	return int(s.difficulty)
}

func (s *stratumV1MinerModel) setOnSubmit(onSubmit interfaces.IHashrate) {
	s.onSubmitMutex.Lock()
	defer s.onSubmitMutex.Unlock()

	s.onSubmit = onSubmit
}

func (s *stratumV1MinerModel) GetWorkerName() string {
	return s.workerName
}

func (s *stratumV1MinerModel) GetConnectedAt() time.Time {
	return s.minerConn.GetConnectedAt()
}

func (s *stratumV1MinerModel) setWorkerName(name string) {
	s.workerName = name
}

func (s *stratumV1MinerModel) RangeDestConn(f func(key any, value any) bool) {
	s.poolConn.RangeConn(f)
}

func (s *stratumV1MinerModel) Cleanup() {
	if s.poolConn != nil {
		err := s.poolConn.Close()
		if err != nil {
			s.log.Errorf("cannot close pool connection %s", err)
		}
	}
	s.onSubmit = nil
}
