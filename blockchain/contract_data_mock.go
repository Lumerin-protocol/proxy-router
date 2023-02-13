package blockchain

import (
	"time"

	"gitlab.com/TitanInd/hashrouter/lib"
)

func GetSampleContractData() ContractData {
	return ContractData{
		Addr:                   lib.GetRandomAddr(),
		Buyer:                  lib.GetRandomAddr(),
		Seller:                 lib.GetRandomAddr(),
		State:                  ContractBlockchainStateAvailable,
		Price:                  10,
		Limit:                  0,
		Speed:                  10,
		Length:                 int64(100),
		StartingBlockTimestamp: time.Now().Unix(),
		Dest:                   lib.MustParseDest("stratum+tcp://default:dest@pool.io:1234"),
	}
}
