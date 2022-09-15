package contractmanager

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type ContractManager struct {
	// dependencies
	blockchain      *blockchain.EthereumGateway
	log             interfaces.ILogger
	globalScheduler *GlobalSchedulerService

	// configuration parameters
	claimFunds       bool
	sellerAddr       blockchain.BlockchainAddress
	sellerPrivateKey string

	// internal state
	contracts interfaces.ICollection[IContractModel]
}

func NewContractManager(blockchain *blockchain.EthereumGateway, globalScheduler *GlobalSchedulerService, log interfaces.ILogger, contracts interfaces.ICollection[IContractModel], sellerAddr blockchain.BlockchainAddress, sellerPrivateKey string) *ContractManager {
	return &ContractManager{
		blockchain:      blockchain,
		globalScheduler: globalScheduler,
		contracts:       contracts,
		log:             log,

		claimFunds:       false,
		sellerAddr:       sellerAddr,
		sellerPrivateKey: sellerPrivateKey,
	}
}

func (m *ContractManager) Run(ctx context.Context) error {
	err := m.runExistingContracts()
	if err != nil {
		return err
	}
	eventsCh, sub, err := m.blockchain.SubscribeToContractCreatedEvent(ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case e := <-eventsCh:
			eventHex, payloadHex := e.Topics[0].Hex(), e.Topics[1].Hex()

			switch eventHex {
			case blockchain.ContractCreatedHex:
				address := common.HexToAddress(payloadHex)
				err := m.handleContract(ctx, address)
				if err != nil {
					m.log.Error("cannot handle created contract, skipping...", err)
				}
			default:
				m.log.Debugf("ignored clonefactory event %s %s", eventHex, payloadHex)
			}

		case <-ctx.Done():
			sub.Unsubscribe()
			return ctx.Err()

		case err := <-sub.Err():
			m.log.Error("contract created subscription error", err)
			return err
		}
	}
}

func (m *ContractManager) runExistingContracts() error {
	existingContractsAddrs, err := m.blockchain.ReadContracts(m.sellerAddr)
	if err != nil {
		m.log.Error("cannot read contracts", err)
		return err
	}
	for _, existingContractAddr := range existingContractsAddrs {
		err := m.handleContract(context.TODO(), existingContractAddr)
		if err != nil {
			m.log.Errorf("cannot fulfill existing contact, skipping, addr: %s", existingContractAddr.Hash().Hex())
		}
	}

	return nil
}

func (m *ContractManager) handleContract(ctx context.Context, address blockchain.BlockchainAddress) error {
	data, err := m.blockchain.ReadContract(address)
	if err != nil {
		return fmt.Errorf("cannot read created contract %w", err)
	}

	m.log.Infof("handling contract \n%+v", data)
	contract := NewContract(data, m.blockchain, m.globalScheduler, m.log, nil)

	go func() {
		err := contract.Run(ctx)
		m.log.Warn("contract error: ", err)
	}()
	m.contracts.Store(contract)

	return nil
}
