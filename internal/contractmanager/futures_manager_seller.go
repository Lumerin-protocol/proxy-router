package contractmanager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Lumerin-protocol/contracts-go/v3/futures"
	"github.com/Lumerin-protocol/proxy-router/internal/interfaces"
	"github.com/Lumerin-protocol/proxy-router/internal/lib"
	"github.com/Lumerin-protocol/proxy-router/internal/repositories/contracts"
	"github.com/Lumerin-protocol/proxy-router/internal/repositories/subgraph"
	"github.com/Lumerin-protocol/proxy-router/internal/resources"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/sync/errgroup"
)

var (
	ErrDeliveryRangeEnded = errors.New("delivery range ended")
)

type FuturesManagerSeller struct {
	futuresAddr common.Address
	userAddr    common.Address

	contracts *lib.Collection[resources.Contract]

	createContract CreateFuturesContractFn
	blockchain     *contracts.FuturesEthereum
	privKey        string
	subgraph       *subgraph.SubgraphClient
	log            interfaces.ILogger
}

type CreateFuturesContractFn func(terms *contracts.FuturesContract) (resources.Contract, error)

func NewFuturesManagerSeller(privKey string, futuresAddr, userAddr common.Address, blockchain *contracts.FuturesEthereum, subgraph *subgraph.SubgraphClient, contracts *lib.Collection[resources.Contract], createContractFn CreateFuturesContractFn, log interfaces.ILogger) *FuturesManagerSeller {
	return &FuturesManagerSeller{
		privKey:        privKey,
		futuresAddr:    futuresAddr,
		userAddr:       userAddr,
		contracts:      contracts,
		createContract: createContractFn,
		blockchain:     blockchain,
		subgraph:       subgraph,
		log:            log,
	}
}

func (fm *FuturesManagerSeller) Run(ctx context.Context) error {
	fm.log.Infof("futures manager started as seller %s", lib.AddrShort(fm.userAddr.Hex()))

	sub, err := fm.blockchain.CreateFuturesSubscription(ctx, fm.futuresAddr)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	fm.log.Infof("subscribed to futures events at address %s", fm.futuresAddr.Hex())

	for {
		start, end, _, err := fm.blockchain.GetOngoingDeliveryRange(ctx)
		if err != nil {
			return lib.WrapError(fmt.Errorf("can't get ongoing delivery range"), err)
		}

		if time.Now().Before(start) {
			fm.log.Infof("waiting for first delivery range to start at %s", start)
			select {
			case <-ctx.Done():
				fm.log.Infof("context done, stopping futures manager seller")
				return ctx.Err()
			case <-time.After(time.Until(start)):
			}
		}

		err = fm.runDeliveryRange(ctx, start, end, sub)
		if err != nil {
			logFn := fm.log.Errorf
			if errors.Is(err, context.Canceled) {
				logFn = fm.log.Warnf
			}
			logFn("delivery range loop exited: %s", err)
			return err
		}
		fm.log.Infof("delivery range ended")
	}
}

func (fm *FuturesManagerSeller) runDeliveryRange(ctx context.Context, start, end time.Time, sub *lib.Subscription) error {
	fm.log.Infof("delivery range started at %s, proxy started at %s", start, time.Now())

	_contracts, err := fm.subgraph.GetPositionsBySeller(ctx, fm.userAddr, start)
	if err != nil {
		return lib.WrapError(fmt.Errorf("can't get contract ids"), err)
	}
	fm.log.Infof("found %d contract units for seller %s at %s", len(_contracts), lib.AddrShort(fm.userAddr.Hex()), start.Format(time.RFC3339))

	errGroup, ctx := errgroup.WithContext(ctx)

	// add contracts to errgroup
	for _, contract := range _contracts {
		if contract.Paid {
			fm.AddContract(ctx, &contract, errGroup)
		} else {
			fm.log.Infof("contract %s is not paid, skipping", lib.AddrShort(contract.ID()))
		}
	}

	// wait for delivery range to end, if it ends, context is cancelled and all errgroup tasks are cancelled
	errGroup.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Until(end)):
			return ErrDeliveryRangeEnded
		}
	})

	// handle events from subscription
	errGroup.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case event := <-sub.Events():
				err := fm.eventController(ctx, event)
				if err != nil {
					fm.log.Errorf("error handling event: %s", err)
				}
			case err := <-sub.Err():
				return err
			}
		}
	})

	err = errGroup.Wait()
	if errors.Is(err, ErrDeliveryRangeEnded) {
		return nil
	}
	return err
}

func (fm *FuturesManagerSeller) tryClaimReward(ctx context.Context, deliveryAt time.Time) {
	err := fm.blockchain.ClaimReward(ctx, deliveryAt, fm.privKey)
	if err != nil {
		fm.log.Errorf("error claiming reward: %s", err)
	}
}

func (fm *FuturesManagerSeller) AddContract(ctx context.Context, data *contracts.FuturesContract, errGroup *errgroup.Group) {
	_, ok := fm.contracts.Load(data.ID())
	if ok {
		fm.log.Errorw("contract already exists in store", "CtrAddr", lib.AddrShort(data.ID()))
		return
	}

	cntr, err := fm.createContract(data)
	if err != nil {
		fm.log.Errorw("contract factory error", "err", err, "CtrAddr", lib.AddrShort(data.ID()))
		return
	}

	fm.contracts.Store(cntr)

	errGroup.Go(func() error {
		err := cntr.Run(ctx)
		fm.log.Warnw(fmt.Sprintf("exited from contract with error %s", err), "CtrAddr", lib.AddrShort(data.ID()))
		fm.contracts.Delete(cntr.ID())
		return nil
	})
}

func (fm *FuturesManagerSeller) RemoveContract(ctx context.Context, data *contracts.FuturesContract) error {
	ctr, ok := fm.contracts.Load(data.ID())
	if !ok {
		return nil
	}
	err := ctr.SyncState(ctx)
	if err != nil {
		fm.log.Errorf("contract sync state error %s", err)
	}
	err = ctr.Stop(ctx)
	if err != nil {
		fm.log.Errorf("contract stop error %s", err)
	} else {
		fm.log.Debugf("contract stopped: %s", lib.AddrShort(ctr.ID()))
	}
	fm.contracts.Delete(ctr.ID())
	fm.log.Debugf("contract deleted: %s", lib.AddrShort(ctr.ID()))
	return nil
}

func (fm *FuturesManagerSeller) eventController(ctx context.Context, event interface{}) error {
	switch e := event.(type) {
	case *futures.FuturesPositionDeliveryClosed:
		return fm.handlePositionDeliveryClosed(ctx, e)
	}
	return nil
}

func (fm *FuturesManagerSeller) handlePositionDeliveryClosed(ctx context.Context, event *futures.FuturesPositionDeliveryClosed) error {
	contract, err := fm.blockchain.GetPosition(ctx, event.PositionId)
	if err != nil {
		return err
	}
	fm.log.Debugf("received position delivery closed event: %s", lib.AddrShort(common.Hash(event.PositionId).Hex()))
	return fm.RemoveContract(ctx, contract)
}

func (fm *FuturesManagerSeller) GetContracts() *lib.Collection[resources.Contract] {
	return fm.contracts
}

func (fm *FuturesManagerSeller) GetContract(id string) (resources.Contract, bool) {
	return fm.contracts.Load(id)
}
