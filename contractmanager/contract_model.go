package contractmanager

import (
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
)

type Contract struct {
	Logger                 interfaces.ILogger
	EthereumGateway        interfaces.IBlockchainGateway
	ContractsGateway       interfaces.IContractsGateway
	IsSeller               bool
	ID                     string
	State                  string
	Buyer                  string
	Price                  int
	Limit                  int
	Speed                  int
	Length                 int
	StartingBlockTimestamp int
	Dest                   string

	fromAddress      interop.BlockchainAddress
	privateKeyString string
	contractAddress  interop.BlockchainAddress
	CurrentNonce     *nonce
	closeOutType     uint
	NodeOperator     *NodeOperator
}

func (c *Contract) GetCurrentNonce() uint64 {
	return c.CurrentNonce.nonce
}
func (c *Contract) GetBuyerAddress() string {
	return c.Buyer
}

func (c *Contract) Execute() (interfaces.IContractModel, error) {
	panic("Contract.Execute not implemented")
	// return c, nil
}

func (c *Contract) GetId() string {
	return c.ID
}

func (c *Contract) SetId(id string) interfaces.IBaseModel {
	newContract := *c

	newContract.ID = id

	return &newContract
}

func (c *Contract) GetCloseOutType() uint {
	return c.closeOutType
}

func (c *Contract) HasDestination() bool {
	return c.Dest != ""
}

func (c *Contract) SetDestination(destination string) {
	c.Dest = destination
}

func (c *Contract) IsAvailable() bool {
	return c.State == ContAvailableState
}

func (c *Contract) SubscribeToContractEvents(address string) (chan interop.BlockchainEvent, interop.BlockchainEventSubscription, error) {
	return c.EthereumGateway.SubscribeToContractEvents(address)
}

func (c *Contract) GetAddress() string {
	return c.ID
}

func (c *Contract) GetPromisedHashrateMin() uint64 {
	panic("Contract.GetPromisedHashrateMin unimplemented")
	return 0
}

func (c *Contract) MakeAvailable() {

	if c.State == ContRunningState {

		c.State = ContAvailableState
		c.Buyer = ""

		c.Save()
	}
}

func (c *Contract) Save() (interfaces.IContractModel, error) {
	return c.ContractsGateway.SaveContract(c)
}

func (c *Contract) GetPrivateKey() string {
	return c.privateKeyString
}

func (c *Contract) TryRunningAt(dest string) (interfaces.IContractModel, error) {
	if c.State == ContRunningState {
		return c.Execute()
	}

	return c, nil
}

var _ interfaces.IContractModel = (*Contract)(nil)
