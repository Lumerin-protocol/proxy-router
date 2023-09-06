package contractmanager

import (
	"context"

	"github.com/Lumerin-protocol/contracts-go/clonefactory"
	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	dataaccess "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/repositories/contracts"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
)

type ContractManager struct {
	contracts       *lib.Collection[resources.Contract]
	contractFactory ContractFactory

	store *dataaccess.HashrateEthereum

	cfAddr common.Address

	log interfaces.ILogger
}

type ContractFactory func(contractData *resources.ContractData) (resources.Contract, error)

func NewContractManager(contractFactory ContractFactory, log interfaces.ILogger) *ContractManager {
	return &ContractManager{
		contracts:       lib.NewCollection[resources.Contract](),
		contractFactory: contractFactory,
		log:             log,
	}
}

func (cm *ContractManager) Run(ctx context.Context) error {
	contractIDs, err := cm.store.GetContractsIDs(ctx)
	if err != nil {
		return err
	}

	for _, id := range contractIDs {
		contractData, err := cm.store.GetContract(ctx, id)
		if err != nil {
			return err
		}
		cm.AddContract(contractData)
	}

	sub, ch, err := cm.store.CreateCloneFactorySubscription(ctx, cm.cfAddr)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-ch:
			err := cm.cloneFactoryController(ctx, event)
			if err != nil {
				return err
			}
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
	contractData, err := cm.store.GetContract(ctx, event.Address.Hex())
	if err != nil {
		return err
	}
	cm.AddContract(contractData)
	return nil
}

func (cm *ContractManager) handleContractPurchased(ctx context.Context, event *clonefactory.ClonefactoryClonefactoryContractPurchased) error {
	return nil
}

func (cm *ContractManager) handleContractDeleteUpdated(ctx context.Context, event *clonefactory.ClonefactoryContractDeleteUpdated) error {
	c, ok := cm.contracts.Load(event.Address.Hex())
	if !ok {
		if !event.IsDeleted {
			contractData, err := cm.store.GetContract(ctx, event.Address.Hex())
			if err != nil {
				return err
			}
			cm.AddContract(contractData)
		}
	}
	if ok {
		cm.log.Info("contract", c)
		if event.IsDeleted {
			// c.StopWatching()
		}
	}
	return nil
}

func (cm *ContractManager) AddContract(data *resources.ContractData) {
	cntr, err := cm.contractFactory(data)
	if err != nil {
		cm.log.Error("contract factory error %s", err)
	}
	cm.contracts.Store(cntr)

	go func() {
		err := cntr.Run(context.TODO())
		if err != nil {
			cm.log.Error("contract ended, error %s", err)
		}
	}()
}

func (cm *ContractManager) GetContracts() *lib.Collection[resources.Contract] {
	return cm.contracts
}
