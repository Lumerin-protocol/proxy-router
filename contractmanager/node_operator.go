package contractmanager

import (
	"gitlab.com/TitanInd/hashrouter/config"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
)

type NodeOperator struct {
	ID                     string
	IsBuyer                bool
	DefaultDest            string
	EthereumAccount        string
	TotalAvailableHashRate int
	UnusedHashRate         int
	Contracts              map[string]string
}

func NewNodeOperator(configuration *config.Config, wallet interfaces.IBlockchainWallet) (*NodeOperator, error) {
	address, err := wallet.GetAddress()

	if err != nil {
		return nil, err
	}

	return &NodeOperator{
		ID:                     interop.NewUniqueIdString(),
		IsBuyer:                configuration.Contract.IsBuyer,
		DefaultDest:            configuration.Pool.Address,
		EthereumAccount:        address.Hex(),
		TotalAvailableHashRate: 0,
		UnusedHashRate:         0,
		Contracts:              make(map[string]string),
	}, nil
}
