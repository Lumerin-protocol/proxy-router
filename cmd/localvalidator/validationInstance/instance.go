package validationInstance

import (
	"example.com/blockHeader"
	"example.com/message"
	"fmt"
	//"math"
	//"errors"
	"math/big"
	"strconv"
	"time"
)

//an individual validator which will operate as a thread
type Validator struct {
	BH               blockHeader.BlockHeader
	StartTime        time.Time
	HashesAnalyzed   uint
	DifficultyTarget uint
	ContractHashRate uint
	ContractLimit    uint
	PoolCredentials  string //the pool username that the hashrate is being allocated to
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
	contractDuration := time.Now().Sub(v.StartTime) //returns an i64
	//calculate the number of hashes represented by the pool difficulty target
	bigDiffTarget := big.NewInt(int64(v.DifficultyTarget))
	bigHashesAnalyzed := big.NewInt(int64(v.HashesAnalyzed))
	hashesPerSubmission := new(big.Int).Exp(big.NewInt(2), bigDiffTarget, nil)
	//multiply the number of hashes checked by the hashes represented from the difficulty target
	totalHashes := new(big.Int).Mul(hashesPerSubmission, bigHashesAnalyzed)
	//divide represented hashes by time duration
	rateBigInt := new(big.Int).Div(totalHashes, big.NewInt(int64(contractDuration.Seconds())))
	return uint(rateBigInt.Uint64())
	//update contracthashrate with value
}

//function to determine if there's any remaining hashrate on the contract
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
func (v *Validator) IncomingHash(credential string, nonce string, time string) (message.HashResult, string) {
	var result = message.HashResult{} //initialize result here to use in error response
	if credential != v.PoolCredentials {
		return result, fmt.Sprintf("Hashrate Hijacking Detected. Check pool user %s", credential)
	}
	calcHash := v.BH.HashInput(nonce, time)                             //calcHash is returned as little endian
	var hashingResult bool                                              //temp until revised logic put in place
	hashAsBigInt, hashingErr := blockHeader.BlockHashToBigInt(calcHash) //designed to intake as little endian
	if hashingErr != nil {
		return result, fmt.Sprintf("error when hashing block: %s", hashingErr)
	}
	var bigDifficulty *big.Int = blockHeader.DifficultyToBigInt(uint32(v.DifficultyTarget))

	if hashAsBigInt.Cmp(bigDifficulty) < 1 {
		hashingResult = true
	} else {
		hashingResult = false
	}
	if hashingResult {
		v.HashesAnalyzed++
	}
	if v.HashrateRemaining() == false {
		v.closeOutContract()
	}
	result.IsCorrect = strconv.FormatBool(hashingResult)
	return result, ""
}

//function to update the validators block header
func (v *Validator) UpdateBlockHeader(bh blockHeader.BlockHeader) {
	v.BH = bh
}
