package protocol

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

type stratumV1MinerModel struct {
	poolConn  StratumV1DestConn
	minerConn StratumV1SourceConn
	validator *hashrate.Hashrate

	difficulty int64
	onSubmit   []OnSubmitHandler
	mutex      sync.RWMutex // guards onSubmit

	configureMsgReq *stratumv1_message.MiningConfigure

	workerName string

	log interfaces.ILogger
}

func NewStratumV1MinerModel(poolPool StratumV1DestConn, miner StratumV1SourceConn, validator *hashrate.Hashrate, log interfaces.ILogger) *stratumV1MinerModel {
	return &stratumV1MinerModel{
		poolConn:  poolPool,
		minerConn: miner,
		validator: validator,
		log:       log,
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
	errCh := make(chan error)
	sendError := func(ctx context.Context, err error) {
		select {
		case errCh <- err:
		case <-subCtx.Done():
		}
	}

	go func() {
		for {
			select {
			case <-subCtx.Done():
				return
			default:
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
			select {
			case <-subCtx.Done():
				return
			default:
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
	close(errCh)

	err = fmt.Errorf("miner model error: %w", err)
	s.log.Error(err)
	return err
}

func (s *stratumV1MinerModel) minerInterceptor(msg stratumv1_message.MiningMessageGeneric) {
	switch typed := msg.(type) {
	case *stratumv1_message.MiningSubmit:
		s.poolConn.RegisterResultHandler(typed.GetID(), func(a stratumv1_message.MiningResult) stratumv1_message.MiningMessageGeneric {
			if a.IsError() {
				s.log.Warnf("error during submit: %s msg ID %d", a.GetError(), a.ID)
				return &a
			}
			s.validator.OnSubmit(s.difficulty)
			s.mutex.RLock()
			defer s.mutex.RUnlock()

			for _, handler := range s.onSubmit {
				handler(uint64(s.difficulty), s.poolConn.GetDest())
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

func (s *stratumV1MinerModel) ChangeDest(dest interfaces.IDestination) error {
	return s.poolConn.SetDest(dest, s.configureMsgReq)
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

func (s *stratumV1MinerModel) GetHashRate() Hashrate {
	return s.validator
}

func (s *stratumV1MinerModel) GetCurrentDifficulty() int {
	return int(s.difficulty)
}

func (s *stratumV1MinerModel) OnSubmit(cb OnSubmitHandler) ListenerHandle {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.onSubmit = append(s.onSubmit, cb)
	return ListenerHandle(len(s.onSubmit))
}

func (s *stratumV1MinerModel) GetWorkerName() string {
	return s.workerName
}

func (s *stratumV1MinerModel) RemoveListener(h ListenerHandle) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.onSubmit[h] = nil
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

func (s *stratumV1MinerModel) Cleanup() error {
	var err error = nil
	if s.poolConn != nil {
		err = s.poolConn.Close()
	}
	s.poolConn = nil
	s.minerConn = nil
	s.validator = nil
	s.onSubmit = nil
	return err
}
