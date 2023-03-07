package contractdata

import (
	"encoding/hex"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
)

type ContractBlockchainState uint8

const (
	ContractBlockchainStateAvailable ContractBlockchainState = iota
	ContractBlockchainStateRunning
)

func (s ContractBlockchainState) String() string {
	switch s {
	case ContractBlockchainStateAvailable:
		return "available"
	case ContractBlockchainStateRunning:
		return "running"
	default:
		return "unknown"
	}
}

type ContractData struct {
	Addr                   common.Address
	Buyer                  common.Address
	Seller                 common.Address
	State                  ContractBlockchainState
	Price                  int64
	Limit                  int64
	Speed                  int64
	Length                 int64
	StartingBlockTimestamp int64
	EncryptedDest          string
}

func NewContractData(addr, buyer, seller common.Address, state uint8, price, limit, speedHS, lengthSeconds, startingBlockTimeUnix int64, encryptedDest string) ContractData {
	return ContractData{
		addr,
		buyer,
		seller,
		ContractBlockchainState(state),
		price,
		limit,
		speedHS,
		lengthSeconds,
		startingBlockTimeUnix,
		encryptedDest,
	}
}

func (c *ContractData) GetID() string {
	return c.GetAddress()
}

func (c *ContractData) GetAddress() string {
	return c.Addr.String()
}

func (c *ContractData) GetBuyerAddress() string {
	return c.Buyer.String()
}

func (c *ContractData) GetSellerAddress() string {
	return c.Seller.String()
}

func (c *ContractData) GetHashrateGHS() int {
	return int(c.Speed / int64(math.Pow10(9)))
}

func (c *ContractData) GetDuration() time.Duration {
	return time.Duration(c.Length) * time.Second
}

func (c *ContractData) GetStartTime() *time.Time {
	startTime := time.Unix(c.StartingBlockTimestamp, 0)
	return &startTime
}

func (c *ContractData) GetEndTime() *time.Time {
	endTime := c.GetStartTime().Add(c.GetDuration())
	return &endTime
}

func (c *ContractData) ContractIsExpired() bool {
	endTime := c.GetEndTime()
	if endTime == nil {
		return false
	}
	return time.Now().After(*endTime)
}

func (c *ContractData) GetStateExternal() string {
	return c.State.String()
}

func (c *ContractData) GetWorkerName() string {
	return c.Addr.String()
}

func (d *ContractData) Copy() ContractData {
	return ContractData{
		Addr:                   d.Addr,
		Buyer:                  d.Buyer,
		Seller:                 d.Seller,
		State:                  d.State,
		Price:                  d.Price,
		Limit:                  d.Limit,
		Speed:                  d.Speed,
		Length:                 d.Length,
		StartingBlockTimestamp: d.StartingBlockTimestamp,
		EncryptedDest:          d.EncryptedDest,
	}
}

func enforceWorkerName(dest lib.Dest, workername string) lib.Dest {
	if dest.String() == "" {
		return lib.Dest{}
	}
	return lib.NewDest(workername, "", dest.GetHost(), nil)
}

type ContractDataDecrypted struct {
	ContractData
	Dest lib.Dest
}

func (c *ContractDataDecrypted) GetDest() interfaces.IDestination {
	return c.Dest
}

func DecryptContractData(contractData ContractData, privateKey string) (ContractDataDecrypted, error) {
	dest, err := decryptDestination(contractData.EncryptedDest, privateKey)
	if err != nil {
		return ContractDataDecrypted{}, err
	}

	destUrl, err := lib.ParseDest(dest)
	if err != nil {
		return ContractDataDecrypted{}, err
	}

	// making sure workername is set to contract ID to be able to identify contract on buyer side
	dst := enforceWorkerName(destUrl, contractData.GetWorkerName())

	return ContractDataDecrypted{
		ContractData: contractData.Copy(),
		Dest:         dst,
	}, nil
}

// decryptDest decrypts destination uri which is encrypted with private key of the contract creator
func decryptDestination(encryptedDestUrl string, privateKey string) (string, error) {
	pkECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return "", err
	}

	pkECIES := ecies.ImportECDSA(pkECDSA)
	destUrlBytes, err := hex.DecodeString(encryptedDestUrl)
	if err != nil {
		return "", err
	}

	decryptedDestUrlBytes, err := pkECIES.Decrypt(destUrlBytes, nil, nil)
	if err != nil {
		return "", err
	}

	return string(decryptedDestUrlBytes), nil
}
