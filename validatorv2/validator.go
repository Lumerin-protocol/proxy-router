package validatorv2

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

// const EMA_INTERVAL = 600
const EMA_INTERVAL = 30

type diffEMA struct {
	diff     int
	lastCalc time.Time
}

type HashResult struct { //string will be true or false
	IsCorrect string
}

//an individual validator which will operate as a thread
type Validator struct {
	BH               BlockHeader
	StartTime        time.Time
	Hashrates        []int
	HashesAnalyzed   uint
	DifficultyTarget uint
	// ContractHashRate uint
	// ContractLimit    uint
	PoolCredentials string //the pool username that the hashrate is being allocated to
	PoolDifficulty  *diffEMA

	MinerID         string
	CurrentHashRate int
	log             interfaces.ILogger
}

func NewValidator(log interfaces.ILogger) *Validator {

	validator := &Validator{
		// BH:               bh,
		StartTime:       time.Now(),
		HashesAnalyzed:  0,
		CurrentHashRate: 0,
		// DifficultyTarget: diff,
		// ContractHashRate: hashRate,
		// ContractLimit:    limit,
		// PoolCredentials:  pc, // pool login credentials
		log: log,
	}
	return validator
}

//emits a message notifying whether or not the block was hashed correctly
func (v *Validator) blockAnalysisMessage(validHash bool) string {
	//revie this to return a JSON message
	if validHash {
		return "block was valid"
	} else {
		return "block was invalid"
	}
}

//need to confirm if DifficultyTarget is already an INT at this point or still in condensed form
func (v *Validator) UpdateHashrate() uint {
	//determine the current duration of the contract
	contractDuration := time.Since(v.StartTime) //returns an i64
	//calculate the number of hashes represented by the pool difficulty target
	bigDiffTarget := big.NewInt(int64(v.DifficultyTarget))
	bigHashesAnalyzed := big.NewInt(int64(v.HashesAnalyzed))

	result := new(big.Int).Exp(big.NewInt(2), big.NewInt(32), nil)
	hashesPerSubmission := new(big.Int).Mul(bigDiffTarget, result)
	totalHashes := new(big.Int).Mul(hashesPerSubmission, bigHashesAnalyzed)

	//divide represented hashes by time duration
	if big.NewInt(int64(contractDuration.Seconds())).Cmp(big.NewInt(0)) == 0 {
		return 0
	} // avoid div by 0 panic
	rateBigInt := new(big.Int).Div(totalHashes, big.NewInt(int64(contractDuration.Seconds())))
	return uint(rateBigInt.Uint64())
	//update contracthashrate with value
}

//function to determine if there's any remaining hashrate on the contract
//this should be deprecated, since the responsibility to determine
//remaining hashrate falls on the lumerin node/contract manager
func (v *Validator) HashrateRemaining() bool {
	return true
	//return v.ContractLimit > v.HashesAnalyzed
}

//function to send message to end contract
//intended to be called by the contract manager. The contract manager will call the smart contract
func (v *Validator) closeOutContract() {
	fmt.Println("web3 call to smart contract to initiate closeout procedure")
}

//receives a nonce and a hash, compares the two, and updates instance parameters
//need to modify to check to see if the resulting hash is below the given difficulty level
//credential is plaintext
//nonce is hex number as a string
//time is hex representation of time
//revise this to return a boolean of true/false for if the hash is deemed to fall below the difficulty target or not
func (v *Validator) IncomingHash(credential string, nonce string, time string) (HashResult, string) {
	var result = HashResult{} //initialize result here to use in error response
	// if credential != v.PoolCredentials {
	// 	return result, fmt.Sprintf("Hashrate Hijacking Detected. Check pool user %s", credential)
	// }
	//calcHash := v.BH.HashInput(nonce, time)                             //calcHash is returned as little endian
	var hashingResult bool //temp until revised logic put in place
	// hashAsBigInt, hashingErr := BlockHashToBigInt(calcHash) //designed to intake as little endian
	// if hashingErr != nil {
	// 	return result, fmt.Sprintf("error when hashing block: %s", hashingErr)
	// }
	// networkDiff := v.BH.Difficulty
	// diff,_ := strconv.ParseUint(networkDiff, 16, 32)
	//var bigDifficulty *big.Int = DifficultyToBigInt(uint32(v.DifficultyTarget + 570425344))

	//if hashAsBigInt.Cmp(bigDifficulty) < 1 {
	hashingResult = true
	//} else {
	//	hashingResult = false
	//}
	if hashingResult {
		v.HashesAnalyzed++
		v.log.Debugf("==========>hashes analyzed: %d", v.HashesAnalyzed)
	}
	if v.HashrateRemaining() == false {
		v.closeOutContract()
	}
	result.IsCorrect = strconv.FormatBool(hashingResult)
	return result, ""
}

func (v *Validator) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// calculate 5 minute moving average of hashrate
		startHashCount := v.HashesAnalyzed
		timeInterval := time.Second * EMA_INTERVAL
		time.Sleep(timeInterval)
		endHashCount := v.HashesAnalyzed
		hashesAnalyzed := endHashCount - startHashCount

		if v.PoolDifficulty == nil {
			v.log.Debugf("pool difficulty not yet defined: skipping hashrate calculation")
			continue
		}

		poolDifficulty := v.PoolDifficulty
		v.log.Debugf("Current Pool Difficulty: %d", poolDifficulty.diff)
		v.log.Debugf("Current Hashes Analyzed in this interval: %d", hashesAnalyzed)

		//calculate the number of hashes represented by the pool difficulty target
		bigDiffTarget := big.NewInt(int64(poolDifficulty.diff))
		bigHashesAnalyzed := big.NewInt(int64(hashesAnalyzed))

		result := new(big.Int).Exp(big.NewInt(2), big.NewInt(32), nil)
		hashesPerSubmission := new(big.Int).Mul(bigDiffTarget, result)
		totalHashes := new(big.Int).Mul(hashesPerSubmission, bigHashesAnalyzed)

		//divide represented hashes by time duration
		rateBigInt := new(big.Int).Div(totalHashes, big.NewInt(int64(timeInterval.Seconds())))
		hashrate := int(rateBigInt.Int64())

		// take average hourly average of hashrate
		if len(v.Hashrates) >= 6 {
			v.Hashrates = v.Hashrates[1:]
		}
		hashSum := 0
		for _, h := range v.Hashrates {
			hashSum += h
		}
		hashSum += hashrate
		newHashrate := hashSum / (len(v.Hashrates) + 1)
		v.Hashrates = append(v.Hashrates, newHashrate)

		v.log.Debugf("current Hashrate Moving Average for Miner %s: %d", v.MinerID, newHashrate)
	}
}

func (v *Validator) GetHashrate() int {
	return v.Hashrates[len(v.Hashrates)-1]
}

func (v *Validator) SetNewDiff(diff int) {
	if v.PoolDifficulty == nil {
		v.PoolDifficulty = &diffEMA{
			diff:     diff,
			lastCalc: time.Now(),
		}
		return
	}

	timePassed := time.Since(v.PoolDifficulty.lastCalc).Seconds()
	timeRatio := timePassed / EMA_INTERVAL

	alpha := 1 - 1.0/math.Exp(timeRatio)
	r := int(alpha*float64(diff) + (1-alpha)*float64(v.PoolDifficulty.diff))

	v.PoolDifficulty.diff = r
	v.PoolDifficulty.lastCalc = time.Now()
}

func (v *Validator) OnMinerNotify(version, previousBlockHash, nBits string, time string, merkelBranches []interface{}) {
	merkelBranchesStr := []string{}
	for _, m := range merkelBranches {
		merkelBranchesStr = append(merkelBranchesStr, m.(string))
	}

	merkelRootStr := ""
	if len(merkelBranchesStr) != 0 {
		merkelRoot, err := ConvertMerkleBranchesToRoot(merkelBranchesStr)
		if err != nil {
			v.log.Panicf("failed to convert merkel branches to merkel root: %w", err)
		}
		merkelRootStr = merkelRoot.String()
	}

	blockHeader := BlockHeader{
		Version:           version,
		PreviousBlockHash: previousBlockHash,
		MerkleRoot:        merkelRootStr,
		Time:              time,
		Difficulty:        nBits,
	}

	v.BH = blockHeader
}
