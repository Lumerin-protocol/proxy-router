package contract

import (
	cm "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/contractmanager"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
)

func NewContractWatcherBuyer(data *cm.ContractData, allocator *allocator.Allocator, log interfaces.ILogger) *ContractWatcher {
	return &ContractWatcher{
		data:       data,
		state:      cm.ContractStatePending,
		allocator:  allocator,
		fullMiners: []string{},
		// actualHRGHS: *hashrate.NewHashrate(),
		log: log,
	}
}
