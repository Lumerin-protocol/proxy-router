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

type FuturesManagerSeller struct {
	futuresAddr common.Address
	userAddr    common.Address

	contracts   *lib.Collection[resources.Contract]
	contractsWG sync.WaitGroup

	createContract CreateFuturesContractFn
	blockchain     *contracts.FuturesEthereum
	subgraph       *subgraph.SubgraphClient
	log            interfaces.ILogger
}

type CreateFuturesContractFn func(terms *contracts.FuturesContract) (resources.Contract, error)

func NewFuturesManagerSeller(futuresAddr, userAddr common.Address, blockchain *contracts.FuturesEthereum, subgraph *subgraph.SubgraphClient, contracts *lib.Collection[resources.Contract], createContractFn CreateFuturesContractFn, log interfaces.ILogger) *FuturesManagerSeller {
	return &FuturesManagerSeller{
		futuresAddr:    futuresAddr,
		userAddr:       userAddr,
		contracts:      contracts,
		createContract: createContractFn,
		blockchain:     blockchain,
		subgraph:       subgraph,
		contractsWG:    sync.WaitGroup{},
		log:            log,
	}
}

func (fm *FuturesManagerSeller) Run(ctx context.Context) error {
	defer func() {
		fm.log.Info("waiting for all contracts to stop")
		fm.contractsWG.Wait()
		fm.log.Info("all contracts stopped")
	}()

	start, _, _, err := fm.blockchain.GetOngoingDeliveryRange(ctx)
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

	fm.log.Infof("delivery range %s started at %s", start.Format(time.RFC3339), time.Now().Format(time.RFC3339))

	_contracts, err := fm.subgraph.GetPositionsBySeller(ctx, fm.userAddr, start)
	if err != nil {
		return lib.WrapError(fmt.Errorf("can't get contract ids"), err)
	}

	fm.log.Infof("found %d contract units for seller %s at %s", len(_contracts), fm.userAddr.Hex(), start.Format(time.RFC3339))

	for _, contract := range _contracts {
		if contract.Paid {
			fm.AddContract(ctx, &contract)
		} else {
			fm.log.Infof("contract %s is not paid, skipping", lib.AddrShort(contract.ID()))
		}
	}

	sub, err := fm.blockchain.CreateFuturesSubscription(ctx, fm.futuresAddr)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	fm.log.Infof("subscribed to futures events at address %s", fm.futuresAddr.Hex())

	for {
		select {
		case <-ctx.Done():
			fm.log.Infof("waiting for all contracts to stop")
			fm.contractsWG.Wait()
			fm.log.Infof("all contracts stopped")
			return nil
		case event := <-sub.Events():
			err := fm.eventController(ctx, event)
			if err != nil {
				return err
			}
		case err := <-sub.Err():
			return err
		}
	}
}

func (fm *FuturesManagerSeller) AddContract(ctx context.Context, data *contracts.FuturesContract) {
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

	fm.contractsWG.Go(func() {
		err := cntr.Run(ctx)
		fm.log.Warnw(fmt.Sprintf("exited from contract %s", err), "CtrAddr", lib.AddrShort(data.ID()))
		fm.contracts.Delete(cntr.ID())
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
	}
	fm.contracts.Delete(ctr.ID())
	return nil
}

func (fm *FuturesManagerSeller) eventController(ctx context.Context, event interface{}) error {
	switch e := event.(type) {
	// case *futures.FuturesPositionPaid:
	// 	return fm.handlePositionPaid(ctx, e)
	// case *futures.FuturesPositionClosed:
	// 	return fm.handlePositionClosed(ctx, e)
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
	return fm.RemoveContract(ctx, contract)
}

// func (fm *FuturesManagerSeller) handlePositionPaid(ctx context.Context, event *futures.FuturesPositionPaid) error {
// contract, err := fm.blockchain.GetPosition(ctx, event.PositionId)
// if err != nil {
// 	return err
// }
// fm.AddContract(ctx, contract)
// return nil
// }

// func (fm *FuturesManagerSeller) handlePositionClosed(ctx context.Context, event *futures.FuturesPositionClosed) error {
// 	contract, err := fm.blockchain.GetPosition(ctx, event.PositionId)
// 	if err != nil {
// 		return err
// 	}
// 	return fm.RemoveContract(ctx, contract)
// }

func (fm *FuturesManagerSeller) GetContracts() *lib.Collection[resources.Contract] {
	return fm.contracts
}

func (fm *FuturesManagerSeller) GetContract(id string) (resources.Contract, bool) {
	return fm.contracts.Load(id)
}
