package contractdata

import (
	"math"
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
		Speed:                  int64(10 * math.Pow10(9)),
		Length:                 int64(100),
		StartingBlockTimestamp: time.Now().Unix(),
		Dest:                   lib.MustParseDest("stratum+tcp://default:dest@pool.io:1234"),
	}
}
