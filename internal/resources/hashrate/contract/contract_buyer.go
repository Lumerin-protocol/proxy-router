package contract

import (
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
)

func NewContractWatcherBuyer(data *resources.ContractData, allocator *allocator.Allocator, log interfaces.ILogger) *ContractWatcher {
	return &ContractWatcher{
		data:       data,
		state:      resources.ContractStatePending,
		allocator:  allocator,
		fullMiners: []string{},
		// actualHRGHS: *hashrate.NewHashrate(),
		log: log,
	}
}
