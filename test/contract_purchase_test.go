package main

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/Lumerin-protocol/contracts-go/clonefactory"
	"github.com/Lumerin-protocol/contracts-go/implementation"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"gitlab.com/TitanInd/hashrouter/blockchain"
	"gitlab.com/TitanInd/hashrouter/constants"
	"gitlab.com/TitanInd/hashrouter/lib"
)

// Local
// var sellerAddress = common.HexToAddress("0xa7af817696d307495ee9efa2ED40fa3Fb9279748")
// var sellerPrivateKey = "b9d76e399dec6f9ba620270a1434236ffdfb37cce2acb32258b1337d3b224a1e"

// Dev
var sellerAddress = common.HexToAddress("0x7525960Bb65713E0A0e226EF93A19a1440f1116d")
var sellerPrivateKey = "3b6bdee2016d0803a11bbb0e3d3b8b5f776f3cf0239b2e5bb53bda317b8a2e20"

var buyerAddress = common.HexToAddress("0x0680bfcbd6a9289b1b78e6a1c6b12bbdbae63082")
var buyerPrivateKey = "5abc94590ab910ba6a9480030bfe1b91d67b5117f00077b4a1ea8f4ed0da889c"
var gethNodeAddress = "wss://goerli.infura.io/ws/v3/4b68229d56fe496e899f07c3d41cb08a"

// var clonefactoryAddress common.Address = common.HexToAddress("0x6372689Fd4A94AE550da5Db7B13B9289F4855dDc") // - local testing
var clonefactoryAddress common.Address = common.HexToAddress("0x60EbdC73d89a9f02D1cA0EbcD842650873c4dec2") // - dev environment
// var clonefactoryAddress common.Address = common.HexToAddress("0x702B0b76235b1DAc489094184B7790cAA9A39Aa4") // - staging environment
// var clonefactoryAddress common.Address = common.HexToAddress("0x78347C1b83BE212c63dcF163091Bb402eB05be9E") // - my test clonefactory

var poolUrl = "stratum+tcp://shev8.contract:@0.0.0.0:3334"

var hashrateContractAddress common.Address = common.HexToAddress("0x867486Bf53648F81F346BeC2aA4f11a25EFDbebE")

// 0x4b6cc541CB35F21323077a84EDE6A662155a0A83 0x4b5C5b20B19B301A6c28cD5060114176Cfc191D5 0x9f8a67886345fd46D3163634b57BEC47D8BB2875 0xaA1A80580B5a9586Cd6dfc24D8e94c1E57308d4c 0x3b6fE2c6AcD5B52a703a9653f4af44B1176978f4

func TestHashrateContractCreation(t *testing.T) {
	// hashrate contract params
	price := 0
	limit := 0
	speed := 5 * int(math.Pow10(12)) // 50 TH/s
	length := 2 * 60 * 60            // 2 hours

	log := lib.NewTestLogger()
	client, err := blockchain.NewEthClient(gethNodeAddress, log)
	if err != nil {
		t.Fatal(err)
	}

	err = CreateHashrateContract(client, sellerAddress, sellerPrivateKey, clonefactoryAddress, price, limit, speed, length, clonefactoryAddress)
	if err != nil {
		t.Fatal(err)
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), ethereum.FilterQuery{
		Addresses: []common.Address{clonefactoryAddress},
	}, logs)
	if err != nil {
		t.Fatal(err)
	}

	for {
		select {
		case err := <-sub.Err():
			t.Fatalf("Error::%v", err)
		case event := <-logs:
			if event.Topics[0].Hex() == blockchain.ContractCreatedHex {
				hashrateContractAddress = common.HexToAddress(event.Topics[1].Hex())
				fmt.Printf("Address of created Hashrate Contract: %v\n\n", hashrateContractAddress.Hex())
				return
			}
		}
	}
}

func TestHashrateContractPurchase(t *testing.T) {
	log := lib.NewTestLogger()
	client, err := blockchain.NewEthClient(gethNodeAddress, log)
	if err != nil {
		t.Fatal(err)
	}

	err = PurchaseHashrateContract(client, buyerAddress, buyerPrivateKey, clonefactoryAddress, hashrateContractAddress, buyerAddress, poolUrl)
	if err != nil {
		t.Fatal(err)
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), ethereum.FilterQuery{
		Addresses: []common.Address{hashrateContractAddress},
	}, logs)
	if err != nil {
		t.Fatal(err)
	}

	for {
		select {
		case err := <-sub.Err():
			t.Fatalf("Error::%v", err)
		case event := <-logs:

			if event.Topics[0].Hex() == blockchain.ContractPurchasedHex {
				hashrateContractAddress := common.HexToAddress(event.Topics[1].Hex())
				fmt.Printf("Address of purchased Hashrate Contract: %v\n\n", hashrateContractAddress.Hex())
				return
			}
		}
	}
}

func TestHashrateRunningContractBuyerUpdate(t *testing.T) {
	updatedDest := "stratum+tcp://shev8.contract3:123@stratum.braiins.com:3333"

	log := lib.NewTestLogger()
	client, err := blockchain.NewEthClient(gethNodeAddress, log)
	if err != nil {
		t.Fatal(err)
	}

	opt, err := getOpt(clonefactoryAddress, buyerPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	impl, err := implementation.NewImplementation(hashrateContractAddress, client)
	if err != nil {
		t.Fatal(err)
	}

	_, err = impl.SetUpdateMiningInformation(opt, updatedDest)
	if err != nil {
		t.Fatal(err)
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), ethereum.FilterQuery{
		Addresses: []common.Address{hashrateContractAddress},
	}, logs)
	if err != nil {
		t.Fatal(err)
	}

	for {
		select {
		case err := <-sub.Err():
			t.Fatalf("Error::%v", err)
		case event := <-logs:
			if event.Topics[0].Hex() == blockchain.ContractCipherTextUpdatedHex {
				fmt.Printf("Address of created Hashrate Contract: %s\n\n", string(event.Data))
			}
		}
	}
}

func TestHashrateContractSellerUpdate(t *testing.T) {
	var (
		price  int64 = 0
		limit  int64 = 0
		speed  int64 = 7 * int64(math.Pow10(12)) // 50 TH/s
		length int64 = 6 * 60 * 60
	)

	log := lib.NewTestLogger()
	client, err := blockchain.NewEthClient(gethNodeAddress, log)
	if err != nil {
		t.Fatal(err)
	}

	opt, err := getOpt(clonefactoryAddress, sellerPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	impl, err := implementation.NewImplementation(hashrateContractAddress, client)
	if err != nil {
		t.Fatal(err)
	}

	_, err = impl.SetUpdatePurchaseInformation(opt, big.NewInt(price), big.NewInt(limit), big.NewInt(speed), big.NewInt(length), big.NewInt(int64(constants.CloseoutTypeWithoutClaim)))
	if err != nil {
		t.Fatal(err)
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), ethereum.FilterQuery{
		Addresses: []common.Address{hashrateContractAddress},
	}, logs)
	if err != nil {
		t.Fatal(err)
	}

	for {
		select {
		case err := <-sub.Err():
			t.Fatalf("Error::%v", err)
		case event := <-logs:
			if event.Topics[0].Hex() == blockchain.ContractPurchaseInfoUpdatedHex {
				fmt.Printf("Address of created Hashrate Contract: %s\n\n", string(event.Data))
			}
		}
	}
}

func CreateHashrateContract(client *ethclient.Client,
	fromAddress common.Address,
	privateKeyString string,
	contractAddress common.Address,
	_price int,
	_limit int,
	_speed int,
	_length int,
	_validator common.Address) error {

	time.Sleep(time.Millisecond * 700)

	instance, err := clonefactory.NewClonefactory(contractAddress, client)
	if err != nil {
		return err
	}

	price := big.NewInt(int64(_price))
	limit := big.NewInt(int64(_limit))
	speed := big.NewInt(int64(_speed))
	length := big.NewInt(int64(_length))

	auth, err := getOpt(fromAddress, privateKeyString)
	if err != nil {
		return err
	}

	tx, err := instance.SetCreateNewRentalContract(auth, price, limit, speed, length, _validator, "")
	if err != nil {
		return err
	}

	fmt.Printf("tx sent: %s\n", tx.Hash().Hex())
	return nil
}

func PurchaseHashrateContract(client *ethclient.Client,
	fromAddress common.Address,
	privateKeyString string,
	contractAddress common.Address,
	_hashrateContract common.Address,
	_buyer common.Address,
	poolData string) error {

	time.Sleep(time.Millisecond * 700)

	instance, err := clonefactory.NewClonefactory(contractAddress, client)
	if err != nil {
		return err
	}

	auth, err := getOpt(fromAddress, privateKeyString)
	if err != nil {
		return err
	}

	tx, err := instance.SetPurchaseRentalContract(auth, _hashrateContract, poolData)
	if err != nil {
		return err
	}
	fmt.Printf("tx sent: %s\n\n", tx.Hash().Hex())
	fmt.Printf("Hashrate Contract %s, was purchased by %s\n\n", _hashrateContract, _buyer)
	return nil
}

func getOpt(addr common.Address, privateKeyString string) (*bind.TransactOpts, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyString)
	if err != nil {
		return nil, err
	}

	client, err := blockchain.NewEthClient(gethNodeAddress, &lib.LoggerMock{})
	if err != nil {
		return nil, err
	}

	// nonce, err := client.PendingNonceAt(context.Background(), addr)
	// if err != nil {
	// 	return nil, err
	// }
	// fmt.Println("Nonce: ", nonce)

	// gasPrice, err := client.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	return nil, err
	// }

	chainId, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		return nil, err
	}
	// auth.Nonce = big.NewInt(int64(nonce))
	// auth.Value = big.NewInt(0)      // in wei
	// auth.GasLimit = uint64(6000000) // in units
	// auth.GasPrice = gasPrice

	return auth, nil
}
