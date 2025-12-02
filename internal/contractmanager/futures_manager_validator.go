package contractmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Lumerin-protocol/contracts-go/v3/futures"
	"github.com/Lumerin-protocol/proxy-router/internal/interfaces"
	"github.com/Lumerin-protocol/proxy-router/internal/lib"
	"github.com/Lumerin-protocol/proxy-router/internal/repositories/contracts"
	"github.com/Lumerin-protocol/proxy-router/internal/repositories/subgraph"
	"github.com/Lumerin-protocol/proxy-router/internal/resources"
	"github.com/ethereum/go-ethereum/common"
)

type FuturesManagerValidator struct {
	futuresAddr   common.Address
	validatorAddr common.Address
	privKey       string

	contracts   *lib.Collection[resources.Contract]
	contractsWG sync.WaitGroup

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
	defer func() {
		fm.log.Info("waiting for all contracts to stop")
		fm.contractsWG.Wait()
		fm.log.Info("all contracts stopped")
	}()

	fm.log.Infof("futures manager started as validator %s", fm.validatorAddr.Hex())

	// delivery range loop
	for {
		start, end, _, err := fm.blockchain.GetOngoingDeliveryRange(ctx)
		if err != nil {
			return lib.WrapError(fmt.Errorf("can't get ongoing delivery range"), err)
		}

		if time.Now().Before(start) {
			fm.log.Infof("waiting for first delivery range to start at %s", start)
			select {
			case <-ctx.Done():
				fm.log.Infof("context done, stopping futures manager validator")
				return ctx.Err()
			case <-time.After(time.Until(start)):
			}
		}

		fm.log.Infof("delivery range %s started at %s", start, time.Now())

		_contracts, err := fm.subgraph.GetAllPositions(ctx, start)
		if err != nil {
			return lib.WrapError(fmt.Errorf("can't get contract ids"), err)
		}

		fm.log.Infof("found %d contract units for validator", len(_contracts), fm.validatorAddr.Hex(), start)

		// close all unpaid contracts with blaming the buyer
		closeDeliveryRequests := make([]contracts.CloseDeliveryReq, 0)
		for _, contract := range _contracts {
			if contract.Paid {
				fm.AddContract(ctx, &contract)
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

		sub, err := fm.blockchain.CreateFuturesSubscription(ctx, fm.futuresAddr)
		if err != nil {
			return err
		}
		defer sub.Unsubscribe()
		fm.log.Infof("subscribed to futures events at address %s", fm.futuresAddr.Hex())

	EVENT_LOOP:
		for {
			select {
			case <-ctx.Done():
				return nil
			case event := <-sub.Events():
				err := fm.eventController(ctx, event)
				if err != nil {
					return err
				}
			case err := <-sub.Err():
				return err
			case <-time.After(time.Until(end)):
				fm.log.Infof("delivery range ended", start, end)
				break EVENT_LOOP
			}
		}
	}
}

func (fm *FuturesManagerValidator) AddContract(ctx context.Context, data *contracts.FuturesContract) {
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

	fm.contractsWG.Go(func() {
		err := cntr.Run(ctx)
		fm.log.Warnw(fmt.Sprintf("exited from contract %s", err), "CtrAddr", lib.AddrShort(data.ID()))
		fm.contracts.Delete(cntr.ID())
	})
}

func (fm *FuturesManagerValidator) RemoveContract(ctx context.Context, data *contracts.FuturesContract) error {
	ctr, ok := fm.contracts.Load(data.ID())
	if !ok {
		return nil
	}
	err := ctr.SyncState(ctx)
	if err != nil {
		fm.log.Errorf("contract sync state error %s", err)
	}
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
	contract, err := fm.blockchain.GetPosition(ctx, event.PositionId)
	if err != nil {
		return err
	}
	return fm.RemoveContract(ctx, contract)
}
