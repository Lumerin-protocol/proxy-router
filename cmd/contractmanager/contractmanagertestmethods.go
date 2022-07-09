package contractmanager

import (
	"context"
	// "crypto/ecdsa"
	// "crypto/rand"
	// "errors"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	//"github.com/ethereum/go-ethereum/crypto/ecies"

	"github.com/ethereum/go-ethereum/ethclient"

	"gitlab.com/TitanInd/lumerin/lumerinlib"

	"gitlab.com/TitanInd/lumerin/lumerinlib/clonefactory"
	"gitlab.com/TitanInd/lumerin/lumerinlib/implementation"
	"gitlab.com/TitanInd/lumerin/lumerinlib/lumerintoken"
)

type TestSetup struct {
	EthClient              *ethclient.Client
	NodeEthereumPrivateKey string
	NodeEthereumAccount    common.Address
	ValidatorAddress       common.Address
	ProxyAddress           common.Address
	LumerinAddress         common.Address
	CloneFactoryAddress    common.Address
}

func LoadTestConfiguration(pkg string, filePath string) (map[string]interface{}, error) {
	var data map[string]interface{}
	var err error = nil
	currDir, _ := os.Getwd()
	defer os.Chdir(currDir)

	if err != nil {
		panic(fmt.Errorf("error retrieving config file variable: %s", err))
	}
	file := filepath.Base(filePath)
	filePath = filepath.Dir(filePath)
	os.Chdir(filePath)

	configFile, err := os.Open(file)
	if err != nil {
		return data, err
	}
	defer configFile.Close()
	byteValue, _ := ioutil.ReadAll(configFile)

	err = json.Unmarshal(byteValue, &data)
	return data[pkg].(map[string]interface{}), err
}

func DeployContract(client *ethclient.Client,
	fromAddress common.Address,
	privateKeyString string,
	constructorParams [4]common.Address,
	contract string) (common.Address, *types.Transaction) {
	privateKey, err := crypto.HexToECDSA(privateKeyString)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	time.Sleep(time.Millisecond * 700)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	fmt.Println("Nonce: ", nonce)

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	chainId, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(6000000) // in units
	auth.GasPrice = gasPrice

	fmt.Printf("Gas price while deploying %v contract: %v WEI\n", contract, auth.GasPrice)
	fmt.Printf("Gas limit while deploying %v contract: %v\n", contract, auth.GasLimit)

	gasPriceInt := gasPrice.Int64()
	gasFee := float64(uint64(gasPriceInt) * auth.GasLimit)
	gasFeeEth := gasFee / math.Pow10(18)
	fmt.Printf("Max Gas Fee while deploying %v contract: %v ETH\n", contract, gasFeeEth)

	lmnAddress := constructorParams[0]
	ValidatorAddress := constructorParams[1]

	switch contract {
	case "LumerinToken":
		address, ltransaction, _, err := lumerintoken.DeployLumerintoken(auth, client)
		if err != nil {
			log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		}
		return address, ltransaction
	case "CloneFactory":
		address, cftransaction, _, err := clonefactory.DeployClonefactory(auth, client, lmnAddress, ValidatorAddress, fromAddress)
		if err != nil {
			log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
		}
		return address, cftransaction
	}

	address := common.HexToAddress("0x0")
	return address, nil
}

func CreateHashrateContract(client *ethclient.Client,
	fromAddress common.Address,
	privateKeyString string,
	contractAddress common.Address,
	_price int,
	_limit int,
	_speed int,
	_length int,
	_validator common.Address) {
	privateKey, err := crypto.HexToECDSA(privateKeyString)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	time.Sleep(time.Millisecond * 700)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	fmt.Println("Nonce: ", nonce)

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	chainId, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(6000000) // in units
	auth.GasPrice = gasPrice

	instance, err := clonefactory.NewClonefactory(contractAddress, client)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	price := big.NewInt(int64(_price))
	limit := big.NewInt(int64(_limit))
	speed := big.NewInt(int64(_speed))
	length := big.NewInt(int64(_length))
	tx, err := instance.SetCreateNewRentalContract(auth, price, limit, speed, length, _validator, "")
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	fmt.Printf("tx sent: %s\n", tx.Hash().Hex())
}

func PurchaseHashrateContract(client *ethclient.Client,
	fromAddress common.Address,
	privateKeyString string,
	contractAddress common.Address,
	_hashrateContract common.Address,
	_buyer common.Address,
	poolData string) {
	privateKey, err := crypto.HexToECDSA(privateKeyString)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	time.Sleep(time.Millisecond * 700)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	fmt.Println("Nonce: ", nonce)

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	chainId, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(6000000) // in units
	auth.GasPrice = gasPrice

	instance, err := clonefactory.NewClonefactory(contractAddress, client)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	tx, err := instance.SetPurchaseRentalContract(auth, _hashrateContract, poolData)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	fmt.Printf("tx sent: %s\n\n", tx.Hash().Hex())
	fmt.Printf("Hashrate Contract %s, was purchased by %s\n\n", _hashrateContract, _buyer)
}

func UpdatePurchaseInformation(client *ethclient.Client,
	fromAddress common.Address,
	privateKeyString string,
	contractAddress common.Address,
	_price int,
	_limit int,
	_speed int,
	_length int) {
	privateKey, err := crypto.HexToECDSA(privateKeyString)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	time.Sleep(time.Millisecond * 700)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	fmt.Println("Nonce: ", nonce)

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	chainId, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(6000000) // in units
	auth.GasPrice = gasPrice

	instance, err := implementation.NewImplementation(contractAddress, client)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	price := big.NewInt(int64(_price))
	limit := big.NewInt(int64(_limit))
	speed := big.NewInt(int64(_speed))
	length := big.NewInt(int64(_length))
	closeOutType := big.NewInt(int64(3))
	tx, err := instance.SetUpdatePurchaseInformation(auth, price, limit, speed, length, closeOutType)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	fmt.Printf("tx sent: %s\n\n", tx.Hash().Hex())
	fmt.Printf("Hashrate Contract %s purchase info was updated\n\n", contractAddress)
}

func UpdateCipherText(client *ethclient.Client,
	fromAddress common.Address,
	privateKeyString string,
	contractAddress common.Address,
	_newEncryptedPoolData string) {
	privateKey, err := crypto.HexToECDSA(privateKeyString)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	time.Sleep(time.Millisecond * 700)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	fmt.Println("Nonce: ", nonce)

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	chainId, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(6000000) // in units
	auth.GasPrice = gasPrice

	instance, err := implementation.NewImplementation(contractAddress, client)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	tx, err := instance.SetUpdateMiningInformation(auth, _newEncryptedPoolData)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}
	fmt.Printf("tx sent: %s\n\n", tx.Hash().Hex())
	fmt.Printf("Hashrate Contract %s Cipher Text Updated \n\n", contractAddress)
}

func CreateNewGanacheBlock(ts TestSetup, account common.Address, privateKey string, contractLength int, sleepTime int) {
	time.Sleep(time.Second * time.Duration(contractLength))

	nonce, err := ts.EthClient.PendingNonceAt(context.Background(), account)
	if err != nil {
		log.Fatal(err)
	}

	gasPrice, err := ts.EthClient.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	gasLimit := uint64(6000000)

	unsignedTx := types.NewTransaction(nonce, account, nil, gasLimit, gasPrice, nil)

	//Sign transaction
	privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := types.SignTx(unsignedTx, types.HomesteadSigner{}, privateKeyECDSA)
	if err != nil {
		log.Fatal(err)
	}

	err = ts.EthClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Millisecond * time.Duration(sleepTime))
}

func BeforeEach(configPath string) (ts TestSetup, ltransaction *types.Transaction, cftransaction *types.Transaction) {
	var constructorParams [4]common.Address
	configData, err := LoadTestConfiguration("contract", configPath)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	mnemonic := configData["mnemonic"].(string)
	account, privateKey := HdWalletKeys(mnemonic, 0)
	ts.NodeEthereumAccount = account.Address
	ts.NodeEthereumPrivateKey = privateKey

	var client *ethclient.Client
	client, err = setUpClient(configData["ethNodeAddr"].(string), ts.NodeEthereumAccount)
	if err != nil {
		log.Fatalf("Funcname::%s, Fileline::%s, Error::%v", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	}

	ts.EthClient = client
	ts.ValidatorAddress = common.HexToAddress(configData["validatorAddress"].(string)) // dummy address
	ts.LumerinAddress, ltransaction = DeployContract(ts.EthClient, ts.NodeEthereumAccount, ts.NodeEthereumPrivateKey, constructorParams, "LumerinToken")
	fmt.Println("Lumerin Token Contract Address: ", ts.LumerinAddress)

	constructorParams[0] = ts.LumerinAddress
	constructorParams[1] = ts.ValidatorAddress

	ts.CloneFactoryAddress, cftransaction = DeployContract(ts.EthClient, ts.NodeEthereumAccount, ts.NodeEthereumPrivateKey, constructorParams, "CloneFactory")
	fmt.Println("Clone Factory Contract Address: ", ts.CloneFactoryAddress)

	return ts, ltransaction, cftransaction
}
