package contractdata

import (
	"math"
	"time"

	"gitlab.com/TitanInd/hashrouter/lib"
)

const PrivateKeySample = "1d9b043a8a288b4510a2085129eb343a5df740773b75fb52882e0072f7d75341"
const EncryptedDestSample = "0467fede2841ffe5cbbbb741ffa77d83ccc91b76f33f93c1f8257880f6aa7564e0f81be4d58835db231e44a82dcdeed39424384bb835e2590bcd97b70140f8a819ada142285ad76795885712ee037530146942f1c7d82612e44ecfc6c33f7a1633765a59e6e9337f599df99e14b35cd7949187326eb09b6f4e9647b8042d7c64d8ba8ff62f00e69a0a8324dc5faf4841defc39e0fa"
const DecryptedDestSample = "stratum+tcp://josh:josh@1.1.1.1:1000"

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
		EncryptedDest:          EncryptedDestSample,
	}
}

func GetSampleContractDataDecrypted() ContractDataDecrypted {
	return ContractDataDecrypted{
		ContractData: GetSampleContractData(),
		Dest:         lib.MustParseDest("stratum+tcp://default:dest@pool.io:1234"),
	}
}
