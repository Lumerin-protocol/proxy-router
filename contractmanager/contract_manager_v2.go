package contractmanager

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
	"gitlab.com/TitanInd/hashrouter/lib"
)

type ContractManager struct {
	// dependencies
	blockchain      *blockchain.EthereumGateway
	log             interfaces.ILogger
	globalScheduler *GlobalSchedulerService

	// configuration parameters
	isBuyer          bool
	claimFunds       bool
	walletAddr       interop.BlockchainAddress
	walletPrivateKey string
	defaultDest      lib.Dest

	// internal state
	contracts interfaces.ICollection[IContractModel]
}

func NewContractManager(blockchain *blockchain.EthereumGateway, globalScheduler *GlobalSchedulerService, log interfaces.ILogger, contracts interfaces.ICollection[IContractModel], walletAddr interop.BlockchainAddress, walletPrivateKey string, isBuyer bool, defaultDest lib.Dest) *ContractManager {
	return &ContractManager{
		blockchain:      blockchain,
		globalScheduler: globalScheduler,
		contracts:       contracts,
		log:             log,

		claimFunds:       false,
		walletAddr:       walletAddr,
		walletPrivateKey: walletPrivateKey,
	}
}

func (m *ContractManager) Run(ctx context.Context) error {
	err := m.runExistingContracts()
	if err != nil {
		return err
	}
	eventsCh, sub, err := m.blockchain.SubscribeToCloneFactoryEvents(ctx)
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

				err = m.handleContract(ctx, address)
				if err != nil {
					m.log.Error("cannot handle contract, skipping...", err)
				}

			case blockchain.ClonefactoryContractPurchasedHex:
				address := common.HexToAddress(payloadHex)

				err = m.handleContract(ctx, address)
				if err != nil {
					m.log.Error("cannot handle purchased contract, skipping...", err)
				}

			default:
				m.log.Debugf("ignored clonefactory event %s %s", eventHex, payloadHex)
			}

		case <-ctx.Done():
			sub.Unsubscribe()
			return ctx.Err()

		case err := <-sub.Err():
			m.log.Error("clonefactory event subscription error", err)
			return err
		}
	}
}

func (m *ContractManager) runExistingContracts() error {
	var existingContractsAddrs []common.Address
	var err error
	existingContractsAddrs, err = m.blockchain.ReadContracts(m.walletAddr, m.isBuyer)
	if err != nil {
		m.log.Error("cannot read contracts", err)
		return err
	}

	for _, existingContractAddr := range existingContractsAddrs {
		err := m.handleContract(context.TODO(), existingContractAddr)
		if err != nil {
			m.log.Errorf("cannot handle existing contact, skipping, addr: %s", existingContractAddr.Hash().Hex())
		}
	}

	m.log.Infof("subscribed to (%d) existing contracts", len(existingContractsAddrs))

	return nil
}

func (m *ContractManager) handleContract(ctx context.Context, contractAddr common.Address) error {
	data, err := m.blockchain.ReadContract(contractAddr)
	if err != nil {
		return fmt.Errorf("cannot read created contract %w", err)
	}

	contract := NewContract(data.(blockchain.ContractData), m.blockchain, m.globalScheduler, m.log, nil, m.isBuyer)

	if contract.Ignore(m.walletAddr, m.defaultDest) {
		// contract will be ignored by this node
		return nil
	}

	m.log.Infof("handling contract \n%+v", data)

	go func() {
		err := contract.Run(ctx)
		m.log.Warn("contract error: ", err)
	}()
	m.contracts.Store(contract)

	return nil
}
