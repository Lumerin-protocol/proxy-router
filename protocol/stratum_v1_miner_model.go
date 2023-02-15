package protocol

import (
	"context"
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

	globalSubmitTracker interfaces.SubmitTracker

	configureMsgReq *stratumv1_message.MiningConfigure

	unansweredMsg      sync.WaitGroup // inverted semaphore that counts messages that were unanswered
	pauseMinerReadCh   chan any
	unpauseMinerReadCh chan any
	pausePoolReadCh    chan any
	unpausePoolReadCh  chan any

	submitErrCount atomic.Int32 // counter of successive submit errors, it is the case when set_extranonce is not supported
	submitErrLimit int          // after this amount of errors miner will be considered faulty
	isFaulty       bool         // true when number of successive errors exceeded limit, miner will be exculded from fulfilling contracts
	onFault        func(ctx context.Context)

	workerName string

	log interfaces.ILogger
}

func NewStratumV1MinerModel(poolPool StratumV1DestConn, minerConn StratumV1SourceConn, validator *hashrate.Hashrate, submitErrLimit int, globalSubmitTracker interfaces.SubmitTracker, log interfaces.ILogger) *stratumV1MinerModel {
	if submitErrLimit == 0 {
		submitErrLimit = DEFAULT_SUBMIT_ERR_COUNT_LIMIT
	}

	return &stratumV1MinerModel{
		poolConn:            poolPool,
		minerConn:           minerConn,
		validator:           validator,
		unansweredMsg:       sync.WaitGroup{},
		pauseMinerReadCh:    make(chan any),
		unpauseMinerReadCh:  make(chan any),
		pausePoolReadCh:     make(chan any),
		unpausePoolReadCh:   make(chan any),
		submitErrCount:      atomic.Int32{},
		submitErrLimit:      submitErrLimit,
		isFaulty:            false,
		globalSubmitTracker: globalSubmitTracker,

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

	s.poolConn.ResendRelevantNotifications(ctx)

	subCtx, cancel := context.WithCancel(ctx)
	errCh := make(chan error, 2)
	sendError := func(ctx context.Context, err error) {
		select {
		case errCh <- err:
		case <-subCtx.Done():
		}
	}

	go func() {
		for {
			if s.pauseUnpause(subCtx, s.pausePoolReadCh, s.unpausePoolReadCh) != nil {
				return
			}

			msg, err := s.poolConn.Read(subCtx)
			if err != nil {
				sendError(subCtx, fmt.Errorf("pool read err: %w", err))
				return
			}

			s.poolInterceptor(msg)

			select {
			case <-subCtx.Done():
				return
			default:
			}

			err = s.minerConn.Write(subCtx, msg)
			if err != nil {
				sendError(subCtx, fmt.Errorf("miner write err: %w", err))
				return
			}
		}
	}()

	go func() {
		for {
			if s.pauseUnpause(subCtx, s.pauseMinerReadCh, s.unpauseMinerReadCh) != nil {
				return
			}

			msg, err := s.minerConn.Read(subCtx)
			if err != nil {
				sendError(subCtx, fmt.Errorf("miner read err: %w", err))
				return
			}

			s.minerInterceptor(msg)

			select {
			case <-subCtx.Done():
				return
			default:
			}

			err = s.poolConn.Write(subCtx, msg)
			if err != nil {
				sendError(subCtx, fmt.Errorf("pool write err: %w", err))
				return
			}
		}
	}()

	err = <-errCh
	cancel()

	err = fmt.Errorf("miner model error: %w", err)
	s.log.Error(err)
	return err
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
			s.globalSubmitTracker.OnSubmit(s.workerName)

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
	s.pauseMinerReadCh <- struct{}{}

	s.unansweredMsg.Wait() // waiting for all responses

	s.pausePoolReadCh <- struct{}{}

	err := s.poolConn.SetDest(ctx, dest, s.configureMsgReq)
	if err != nil {
		return err
	}
	s.setOnSubmit(onSubmit)

	s.unpausePoolReadCh <- struct{}{}
	s.unpauseMinerReadCh <- struct{}{}

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

// pauseUnpause pauses execution after receiving signal on pauseChan, and blocks until unpause signal is not received
// It supports context, and on cancellation returns error
func (s *stratumV1MinerModel) pauseUnpause(ctx context.Context, pauseChan, unpauseChan chan any) (err error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-pauseChan:
	INNER:
		for {
			select {
			case <-pauseChan:
				// do nothing, block in the loop
			case <-unpauseChan:
				break INNER
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(MAX_PAUSE_DURATION):
				s.log.Warnf("max pause time reached (%s), unpaused", MAX_PAUSE_DURATION)
				break INNER
			}
		}

	case <-unpauseChan: // unblock
	default:
	}

	return nil
}
