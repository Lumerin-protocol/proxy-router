package contractmanager

import (
	"context"
	"net/url"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
)

type ContractData struct {
	ContractID        string
	Seller            string
	Buyer             string
	Dest              *url.URL
	StartedAt         *time.Time
	Duration          time.Duration
	ContractRole      ContractRole
	ResourceType      ResourceType
	ResourceEstimates map[string]float64 // hashrate
}

type ContractManager struct {
	contracts       *lib.Collection[Contract]
	contractFactory ContractFactory
	log             interfaces.ILogger
}

type ContractFactory func(contractData *ContractData) (Contract, error)

func NewContractManager(contractFactory ContractFactory, log interfaces.ILogger) *ContractManager {
	return &ContractManager{
		contracts:       lib.NewCollection[Contract](),
		contractFactory: contractFactory,
		log:             log,
	}
}

func (cm *ContractManager) AddContract(data *ContractData) {
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

func (cm *ContractManager) GetContracts() *lib.Collection[Contract] {
	return cm.contracts
}
