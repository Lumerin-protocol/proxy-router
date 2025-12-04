package contractmanager

import (
	"context"
	"fmt"
	"sync"

	"github.com/Lumerin-protocol/contracts-go/v2/clonefactory"
	"github.com/Lumerin-protocol/proxy-router/internal/interfaces"
	"github.com/Lumerin-protocol/proxy-router/internal/lib"
	"github.com/Lumerin-protocol/proxy-router/internal/repositories/contracts"
	"github.com/Lumerin-protocol/proxy-router/internal/resources"
	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate"
	"github.com/ethereum/go-ethereum/common"
)

type ContractManager struct {
	cfAddr    common.Address
	ownerAddr common.Address

	contracts   *lib.Collection[resources.Contract]
	contractsWG sync.WaitGroup

	createContract CreateContractFn
	store          *contracts.HashrateEthereum
	log            interfaces.ILogger
}

type CreateContractFn func(terms *hashrate.EncryptedTerms) (resources.Contract, error)

func NewContractManager(clonefactoryAddr, ownerAddr common.Address, createContractFn CreateContractFn, store *contracts.HashrateEthereum, contracts *lib.Collection[resources.Contract], log interfaces.ILogger) *ContractManager {
	return &ContractManager{
		cfAddr:         clonefactoryAddr,
		ownerAddr:      ownerAddr,
		contracts:      contracts,
		createContract: createContractFn,
		store:          store,
		contractsWG:    sync.WaitGroup{},
		log:            log,
	}
}

func (cm *ContractManager) Run(ctx context.Context) error {
	defer func() {
		cm.log.Info("waiting for all contracts to stop")
		cm.contractsWG.Wait()
		cm.log.Info("all contracts stopped")
	}()

	contractIDs, err := cm.store.GetContractsIDs(ctx)
	if err != nil {
		return lib.WrapError(fmt.Errorf("can't get contract ids"), err)
	}

	for _, id := range contractIDs {
		terms, err := cm.store.GetContract(ctx, id)
		if err != nil {
			return lib.WrapError(fmt.Errorf("can't get contract"), err)
		}
		if cm.isOurContract(terms) {
			cm.AddContract(ctx, terms)
		}
	}

	sub, err := cm.store.CreateCloneFactorySubscription(ctx, cm.cfAddr)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	cm.log.Infof("subscribed to clonefactory events at address %s", cm.cfAddr.Hex())

	for {
		select {
		case <-ctx.Done():
			//TODO: wait until all child contracts are stopped
			return nil
		case event := <-sub.Events():
			err := cm.cloneFactoryController(ctx, event)
			if err != nil {
				return err
			}
		case err := <-sub.Err():
			return err
		}
	}
}

func (cm *ContractManager) cloneFactoryController(ctx context.Context, event interface{}) error {
	switch e := event.(type) {
	case *clonefactory.ClonefactoryContractCreated:
		return cm.handleContractCreated(ctx, e)
	case *clonefactory.ClonefactoryClonefactoryContractPurchased:
		return cm.handleContractPurchased(ctx, e)
	case *clonefactory.ClonefactoryContractDeleteUpdated:
		return cm.handleContractDeleteUpdated(ctx, e)
	}
	return nil
}

func (cm *ContractManager) handleContractCreated(ctx context.Context, event *clonefactory.ClonefactoryContractCreated) error {
	terms, err := cm.store.GetContract(ctx, event.Address.Hex())
	if err != nil {
		return err
	}
	if cm.isOurContract(terms) {
		cm.AddContract(ctx, terms)
	}
	return nil
}

func (cm *ContractManager) handleContractPurchased(ctx context.Context, event *clonefactory.ClonefactoryClonefactoryContractPurchased) error {
	cm.log.Debugf("clonefactory contract purchased event, address %s", event.Address.Hex())
	terms, err := cm.store.GetContract(ctx, event.Address.Hex())
	if err != nil {
		return err
	}
	if terms.Buyer() == cm.ownerAddr.String() || terms.Validator() == cm.ownerAddr.String() {
		cm.AddContract(ctx, terms)
	}
	return nil
}

func (cm *ContractManager) handleContractDeleteUpdated(ctx context.Context, event *clonefactory.ClonefactoryContractDeleteUpdated) error {
	ctr, ok := cm.contracts.Load(event.Address.Hex())
	if !ok {
		return nil
	}
	err := ctr.SyncState(ctx)
	if err != nil {
		cm.log.Errorf("contract sync state error %s", err)
	}
	return nil
}

func (cm *ContractManager) AddContract(ctx context.Context, data *hashrate.EncryptedTerms) {
	_, ok := cm.contracts.Load(data.ID())
	if ok {
		cm.log.Errorw("contract already exists in store", "CtrAddr", lib.AddrShort(data.ID()))
		return
	}

	cntr, err := cm.createContract(data)
	if err != nil {
		cm.log.Errorw("contract factory error", "err", err, "CtrAddr", lib.AddrShort(data.ID()))
		return
	}

	cm.contracts.Store(cntr)

	cm.contractsWG.Go(func() {
		err := cntr.Run(ctx)
		cm.log.Warnw(fmt.Sprintf("exited from contract with error %s", err), "CtrAddr", lib.AddrShort(data.ID()))
		cm.contracts.Delete(cntr.ID())
	})
}

func (cm *ContractManager) GetContracts() *lib.Collection[resources.Contract] {
	return cm.contracts
}

func (cm *ContractManager) GetContract(id string) (resources.Contract, bool) {
	return cm.contracts.Load(id)
}

func (cm *ContractManager) isOurContract(terms TermsCommon) bool {
	return terms.Seller() == cm.ownerAddr.String() || terms.Buyer() == cm.ownerAddr.String() || terms.Validator() == cm.ownerAddr.String()
}
