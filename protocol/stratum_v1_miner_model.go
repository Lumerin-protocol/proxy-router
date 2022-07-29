package protocol

import (
	"context"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type stratumV1MinerModel struct {
	pool  StratumV1DestConn
	miner StratumV1SourceConn
	log   interfaces.ILogger
}

func NewStratumV1MinerModel(poolPool StratumV1DestConn, miner StratumV1SourceConn, log interfaces.ILogger) *stratumV1MinerModel {
	return &stratumV1MinerModel{
		pool:  poolPool,
		miner: miner,
		log:   log,
	}
}

func (s *stratumV1MinerModel) Run() error {
	s.log.Info("proxying started")
	errCh := make(chan error)
	go func() {
		for {
			msg, err := s.pool.Read(context.TODO())
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

func (s *stratumV1MinerModel) ChangeDest(addr string, authUser string, authPwd string) error {
	err := s.pool.SetDest(addr, authUser, authPwd)
	return err
}

func (s *stratumV1MinerModel) GetID() string {
	return s.miner.GetID()
}

func (s *stratumV1MinerModel) IsAvailable() bool {
	return true
}
