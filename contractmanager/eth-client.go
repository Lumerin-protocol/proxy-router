package contractmanager

import (
	"github.com/ethereum/go-ethereum/ethclient"
)

func NewEthClient(clientAddress string) (client *ethclient.Client, err error) {
	return ethclient.Dial(clientAddress)
	// if err != nil {
	//fmt.Printf("Funcname::%v, Fileline::%v, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	// 	return client, err
	// }

	//fmt.Printf("Connected to rpc client at %v\n", clientAddress)

	// var balance *big.Int
	// balance, err = client.BalanceAt(context.Background(), contractManagerAccount, nil)
	// if err != nil {
	// 	//fmt.Printf("Funcname::%v, Fileline::%v, Error::%v\n", lumerinlib.Funcname(), lumerinlib.FileLine(), err)
	// 	return client, err
	// }
	// fbalance := new(big.Float)
	// fbalance.SetString(balance.String())
	// ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))

	// fmt.Println("Balance of contract manager account:", ethValue, "ETH")

	// return client, err
}
