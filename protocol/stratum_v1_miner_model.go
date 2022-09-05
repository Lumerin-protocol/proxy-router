package protocol

import (
	"context"
	"sync"

	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

type stratumV1MinerModel struct {
	pool      StratumV1DestConn
	miner     StratumV1SourceConn
	validator *hashrate.Hashrate

	difficulty int64
	onSubmit   []OnSubmitHandler
	mutex      sync.RWMutex // guards onSubmit

	extraNonceMsg *stratumv1_message.MiningSubscribeResult
	workerName    string

	log interfaces.ILogger
}

func NewStratumV1MinerModel(poolPool StratumV1DestConn, miner StratumV1SourceConn, validator *hashrate.Hashrate, log interfaces.ILogger) *stratumV1MinerModel {
	return &stratumV1MinerModel{
		pool:      poolPool,
		miner:     miner,
		validator: validator,
		log:       log,
	}
}

func (s *stratumV1MinerModel) Connect() error {
	for {
		m, err := s.miner.Read(context.TODO())
		if err != nil {
			s.log.Error(err)
			panic(err)
			break
		}

		switch typedMessage := m.(type) {
		case *stratumv1_message.MiningSubscribe:
			extranonce, size := s.pool.GetExtranonce()
			msg := stratumv1_message.NewMiningSubscribeResult(extranonce, size)

			msg.SetID(typedMessage.GetID())
			err := s.miner.Write(context.TODO(), msg)
			if err != nil {
				panic(err)
			}
			continue

		case *stratumv1_message.MiningAuthorize:
			s.setWorkerName(typedMessage.GetWorkerName())

			msg, _ := stratumv1_message.ParseMiningResult([]byte(`{"id":47,"result":true,"error":null}`))
			msg.SetID(typedMessage.GetID())
			err := s.miner.Write(context.TODO(), msg)
			if err != nil {
				panic(err)
			}
			return nil

		case *stratumv1_message.MiningConfigure:
			msg, err := s.pool.SendPoolRequestWait(typedMessage)
			if err != nil {
				panic(err)
			}
			err = s.miner.Write(context.TODO(), msg)
			if err != nil {
				panic(err)
			}
		}
	}
	return nil
}

func (s *stratumV1MinerModel) Run() error {
	err := s.Connect()
	if err != nil {
		panic(err)
	}
	s.pool.ResendRelevantNotifications(context.TODO())

	errCh := make(chan error)
	go func() {
		for {
			msg, err := s.pool.Read(context.TODO())
			if err != nil {
				s.log.Error(err)
				errCh <- err
				return
			}
			s.poolInterceptor(msg)

			err = s.miner.Write(context.TODO(), msg)
			if err != nil {
				s.log.Error(err)
				errCh <- err
				return
			}
		}
	}()

	go func() {
		for {
			msg, err := s.miner.Read(context.TODO())
			if err != nil {
				s.log.Error(err)
				errCh <- err
				return
			}

			s.minerInterceptor(msg)

			err = s.pool.Write(context.TODO(), msg)
			if err != nil {
				s.log.Error(err)
				errCh <- err
				return
			}
		}
	}()

	return <-errCh
}

func (s *stratumV1MinerModel) minerInterceptor(msg stratumv1_message.MiningMessageGeneric) {
	switch msg.(type) {
	case *stratumv1_message.MiningSubmit:
		s.validator.OnSubmit(s.difficulty)

		s.mutex.RLock()
		defer s.mutex.RUnlock()

		for _, handler := range s.onSubmit {
			handler(uint64(s.difficulty), s.pool.GetDest())
		}
	}
}

func (s *stratumV1MinerModel) poolInterceptor(msg stratumv1_message.MiningMessageGeneric) {
	switch m := msg.(type) {
	case *stratumv1_message.MiningSetDifficulty:
		//TODO: some pools return difficulty in float, decide if we need that kind of precision
		s.difficulty = int64(m.GetDifficulty())
	}
}

func (s *stratumV1MinerModel) ChangeDest(dest interfaces.IDestination) error {
	err := s.pool.SetDest(dest)
	return err
}

func (s *stratumV1MinerModel) GetDest() interfaces.IDestination {
	return s.pool.GetDest()
}

func (s *stratumV1MinerModel) GetID() string {
	return s.miner.GetID()
}

func (s *stratumV1MinerModel) GetHashRateGHS() int {
	return s.validator.GetHashrateGHS()
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

func (s *stratumV1MinerModel) setWorkerName(name string) {
	s.workerName = name
}
