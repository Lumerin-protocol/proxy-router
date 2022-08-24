package contractmanager

import (
	"time"

	"gitlab.com/TitanInd/hashrouter/contractmanager/blockchain"
	"gitlab.com/TitanInd/hashrouter/hashrate"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
)

type Contract struct {
	Logger                interfaces.ILogger
	EthereumGateway       interfaces.IBlockchainGateway
	ContractsGateway      interfaces.IContractsGateway
	RoutableStreamService interfaces.IRoutableStreamsService

	IsSeller               bool
	ID                     string
	State                  string
	Buyer                  string
	Price                  int
	Limit                  int
	Speed                  int
	Length                 int
	StartingBlockTimestamp int
	Dest                   interfaces.IDestination

	fromAddress      blockchain.BlockchainAddress
	contractAddress  blockchain.BlockchainAddress
	privateKeyString string
	CurrentNonce     *nonce
	closeOutType     uint
	NodeOperator     *NodeOperator

	hashrate    *hashrate.Hashrate // the counter of single contract
	combination HashrateList       // combination of full/partially allocated miners fulfilling the contract
}

func NewContract(combination HashrateList, hashrate *hashrate.Hashrate) *Contract {
	return &Contract{
		combination: combination,
		hashrate:    hashrate,
	}
}

func (c *Contract) Run() {
	for {
		// TODO: routine that checks whether the hashrate is fulfilled
		// if not it replaces miner
		//
		// check if contract duration is finished
		time.Sleep(time.Minute)
	}
}

func (c *Contract) GetCurrentNonce() uint64 {
	return c.CurrentNonce.nonce
}

func (c *Contract) GetBuyerAddress() string {
	return c.Buyer
}

func (c *Contract) SetBuyerAddress(buyer string) {
	c.Buyer = buyer
}

func (c *Contract) Initialize() (interfaces.ISellerContractModel, error) {
	panic("Unimplemented method: Contract.Initialize")
}

func (c *Contract) Execute() (interfaces.ISellerContractModel, error) {
	c.Logger.Debugf("Executing contract %v", c.GetID())
	c.RoutableStreamService.ChangeDestAll(c.Dest)
	c.Logger.Debugf("Changed destination to %v", c.Dest.String())

	return c, nil
}

func (c *Contract) GetID() string {
	return c.ID
}

func (c *Contract) SetID(id string) interfaces.IBaseModel {
	newContract := *c

	newContract.ID = id

	return &newContract
}

func (c *Contract) GetCloseOutType() uint {
	return c.closeOutType
}

func (c *Contract) SetDestination(dest string) (err error) {
	c.Dest, err = lib.ParseDest(dest)

	if err != nil {
		return err
	}

	return nil
}

func (c *Contract) IsAvailable() bool {
	return c.State == ContAvailableState
}

func (c *Contract) GetAddress() string {
	return c.ID
}

func (c *Contract) GetHashrateGHS() int {
	return c.Speed / 1000
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

func (c *Contract) Save() (interfaces.ISellerContractModel, error) {
	return c.ContractsGateway.SaveContract(c)
}

func (c *Contract) GetPrivateKey() string {
	return c.privateKeyString
}

func (c *Contract) TryRunningAt(dest string) (interfaces.ISellerContractModel, error) {
	if c.State == ContRunningState {
		return c.Execute()
	}

	return c, nil
}

var _ interfaces.ISellerContractModel = (*Contract)(nil)
