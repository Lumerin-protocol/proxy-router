package blockchain

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

func NewEthClient(clientAddress string, log interfaces.ILogger) (client *ethclient.Client, err error) {
	return ethclient.Dial(clientAddress)
}
