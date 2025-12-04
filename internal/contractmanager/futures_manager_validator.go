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

type FuturesManagerValidator struct {
	futuresAddr   common.Address
	validatorAddr common.Address
	privKey       string

	contracts *lib.Collection[resources.Contract]

	createContractValidator CreateFuturesContractFn
	blockchain              *contracts.FuturesEthereum
	subgraph                *subgraph.SubgraphClient
	log                     interfaces.ILogger
}

func NewFuturesManagerValidator(privKey string, futuresAddr, validatorAddr common.Address, blockchain *contracts.FuturesEthereum, subgraph *subgraph.SubgraphClient, contracts *lib.Collection[resources.Contract], createContractValidator CreateFuturesContractFn, log interfaces.ILogger) *FuturesManagerValidator {
	return &FuturesManagerValidator{
		privKey:                 privKey,
		futuresAddr:             futuresAddr,
		validatorAddr:           validatorAddr,
		contracts:               contracts,
		createContractValidator: createContractValidator,
		blockchain:              blockchain,
		subgraph:                subgraph,
		log:                     log,
	}
}

func (fm *FuturesManagerValidator) Run(ctx context.Context) error {
	fm.log.Infof("futures manager started as validator %s", lib.AddrShort(fm.validatorAddr.Hex()))

	sub, err := fm.blockchain.CreateFuturesSubscription(ctx, fm.futuresAddr)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	fm.log.Infof("subscribed to futures events at address %s", fm.futuresAddr.Hex())

	// delivery range loop
	for {
		deliveryAt, end, _, err := fm.blockchain.GetOngoingDeliveryRange(ctx)
		if err != nil {
			return lib.WrapError(fmt.Errorf("can't get ongoing delivery range"), err)
		}

		if time.Now().Before(deliveryAt) {
			fm.log.Infof("waiting for first delivery range to start at %s", deliveryAt)
			select {
			case <-ctx.Done():
				fm.log.Infof("context done, stopping futures manager validator")
				return ctx.Err()
			case <-time.After(time.Until(deliveryAt)):
			}
		}

		err = fm.runDeliveryRange(ctx, deliveryAt, end, sub)
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

func (fm *FuturesManagerValidator) runDeliveryRange(ctx context.Context, deliveryAt time.Time, end time.Time, sub *lib.Subscription) error {
	fm.log.Infof("delivery range started at %s, proxy started at %s", deliveryAt, time.Now())

	_contracts, err := fm.subgraph.GetAllPositions(ctx, deliveryAt)
	if err != nil {
		return lib.WrapError(fmt.Errorf("can't get contract ids"), err)
	}

	fm.log.Infof("found %d contract units for validator", len(_contracts))

	errGroup, ctx := errgroup.WithContext(ctx)

	// close all unpaid contracts with blaming the buyer
	paidContracts := make([]*contracts.FuturesContract, 0)
	closeDeliveryRequests := make([]contracts.CloseDeliveryReq, 0)
	for _, contract := range _contracts {
		if contract.Paid {
			paidContracts = append(paidContracts, &contract)
		} else {
			closeDeliveryRequests = append(closeDeliveryRequests, contracts.CloseDeliveryReq{
				PositionID:  contract.ContractID,
				BlameSeller: false,
			})
		}
	}

	fm.log.Infof("found %d unpaid contracts", len(closeDeliveryRequests))
	if len(closeDeliveryRequests) > 0 {
		err = fm.blockchain.BatchCloseDelivery(ctx, closeDeliveryRequests, fm.privKey)
		if err != nil {
			return lib.WrapError(fmt.Errorf("can't close delivery"), err)
		}
		fm.log.Infof("closed %d unpaid contracts", len(closeDeliveryRequests))
	}

	// add paid contracts to errgroup
	for _, contract := range paidContracts {
		fm.AddContract(ctx, contract, errGroup)
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

func (fm *FuturesManagerValidator) AddContract(ctx context.Context, data *contracts.FuturesContract, errGroup *errgroup.Group) {
	_, ok := fm.contracts.Load(data.ID())
	if ok {
		fm.log.Errorw("contract already exists in store", "CtrAddr", lib.AddrShort(data.ID()))
		return
	}

	cntr, err := fm.createContractValidator(data)
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
	fm.log.Infof("waiting for contract to start: %s", cntr.ID())
	<-cntr.Started()
	fm.log.Infof("contract started: %s", cntr.ID())
}

func (fm *FuturesManagerValidator) RemoveContract(ctx context.Context, positionID common.Hash) error {
	ctr, ok := fm.contracts.Load(positionID.Hex())
	if !ok {
		fm.log.Debugf("contract not found: %s, ignoring", positionID.Hex())
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
		fm.log.Debugf("contract stopped: %s", ctr.ID())
	}
	fm.contracts.Delete(ctr.ID())
	fm.log.Debugf("contract deleted: %s", ctr.ID())
	return nil
}

func (fm *FuturesManagerValidator) eventController(ctx context.Context, event interface{}) error {
	switch e := event.(type) {
	case *futures.FuturesPositionDeliveryClosed:
		return fm.handlePositionDeliveryClosed(ctx, e)
	}
	return nil
}

func (fm *FuturesManagerValidator) handlePositionDeliveryClosed(ctx context.Context, event *futures.FuturesPositionDeliveryClosed) error {
	fm.log.Debugf("received position delivery closed event: %s", lib.AddrShort(common.Hash(event.PositionId).Hex()))
	return fm.RemoveContract(ctx, event.PositionId)
}
