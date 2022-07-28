package protocol

import (
	"context"

	"gitlab.com/TitanInd/hashrouter/protocol/message"
	"go.uber.org/zap"
)

type StratumV1MinerInterface interface {
	GetID() string
	Read(ctx context.Context) (message.MiningMessageGeneric, error)
	Write(ctx context.Context, msg message.MiningMessageGeneric) error
}

type stratumManagerV2 struct {
	pool  *StratumV1PoolPool
	miner StratumV1MinerInterface
	log   *zap.SugaredLogger
}

func NewStratumManagerV2(poolPool *StratumV1PoolPool, miner StratumV1MinerInterface, log *zap.SugaredLogger) *stratumManagerV2 {
	return &stratumManagerV2{
		pool:  poolPool,
		miner: miner,
		log:   log,
	}
}

func (s *stratumManagerV2) Run() error {
	s.log.Info("proxying started")
	errCh := make(chan error)
	go func() {
		for {
			msg, err := s.pool.GetConn().Read()
			if err != nil {
				s.log.Error(err)
				errCh <- err
				return
			}
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

			err = s.pool.GetConn().Write(context.TODO(), msg)
			if err != nil {
				s.log.Error(err)
				errCh <- err
				return
			}
		}
	}()

	return <-errCh
}

func (s *stratumManagerV2) ChangePool(addr string, authUser string, authPwd string) error {
	err := s.pool.SetDest(addr, authUser, authPwd)
	return err
}

func (s *stratumManagerV2) GetID() string {
	return s.miner.GetID()
}
