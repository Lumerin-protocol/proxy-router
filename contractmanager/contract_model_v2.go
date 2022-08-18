package contractmanager

import (
	"time"

	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
	"gitlab.com/TitanInd/hashrouter/miner"
)

type ContractV2 struct {
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

	minerSchedulers []miner.MinerScheduler // holds references to single miner schedulers which at least partially fulfilling the contract
	hashrate        hashrate.Hashrate      // the counter of single contract
}

func (c *ContractV2) NewContractV2(minerSchedulers []miner.MinerScheduler) *ContractV2 {
	return &ContractV2{
		minerSchedulers: minerSchedulers,
		hashrate:        *hashrate.NewHashrate(c.Logger, hashrate.EMA_INTERVAL),
	}

	//TODO: start listening to events from minerSchedulers (i.e. submit, disconnect)
	// consider wiring event handlers in service code
}

func (c *ContractV2) OnMinerSubmit(diff int64) {
	c.hashrate.OnSubmit(diff)
}

func (c *ContractV2) OnMinerDisconnect(diff int64) {
	// assign different miner for contract
}

func (c *ContractV2) Run() {
	for {
		// TODO: routine that checks whether the hashrate is fulfilled
		// if not it replaces miner
		//
		// check if contract duration is finished
		time.Sleep(time.Minute)
	}
}

func (c *ContractV2) GetCurrentNonce() uint64 {
	return c.CurrentNonce.nonce
}
func (c *ContractV2) GetBuyerAddress() string {
	return c.Buyer
}

func (c *ContractV2) Execute() (interfaces.IContractModel, error) {
	panic("Contract.Execute not implemented")
	// return c, nil
}

func (c *ContractV2) GetId() string {
	return c.ID
}

func (c *ContractV2) SetId(id string) interfaces.IBaseModel {
	newContract := *c

	newContract.ID = id

	return &newContract
}

func (c *ContractV2) GetCloseOutType() uint {
	return c.closeOutType
}

func (c *ContractV2) HasDestination() bool {
	return c.Dest != ""
}

func (c *ContractV2) SetDestination(destination string) {
	c.Dest = destination
}

func (c *ContractV2) IsAvailable() bool {
	return c.State == ContAvailableState
}

func (c *ContractV2) SubscribeToContractEvents(address string) (chan interop.BlockchainEvent, interop.BlockchainEventSubscription, error) {
	return c.EthereumGateway.SubscribeToContractEvents(address)
}

func (c *ContractV2) GetAddress() string {
	return c.ID
}

func (c *ContractV2) GetPromisedHashrateMin() uint64 {
	panic("Contract.GetPromisedHashrateMin unimplemented")
	return 0
}

func (c *ContractV2) MakeAvailable() {

	if c.State == ContRunningState {

		c.State = ContAvailableState
		c.Buyer = ""

		c.Save()
	}
}

func (c *ContractV2) Save() (interfaces.IContractModel, error) {
	return c.ContractsGateway.SaveContract(c)
}

func (c *ContractV2) GetPrivateKey() string {
	return c.privateKeyString
}

func (c *ContractV2) TryRunningAt(dest string) (interfaces.IContractModel, error) {
	if c.State == ContRunningState {
		return c.Execute()
	}

	return c, nil
}

var _ interfaces.IContractModel = (*Contract)(nil)
