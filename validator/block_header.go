package validator

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	chainhashOnline "github.com/btcsuite/btcd/chaincfg/chainhash"
	wire "github.com/btcsuite/btcd/wire"
)

type BlockHeader struct {
	Version           string // little endian hex format
	PreviousBlockHash string // little endian hex format
	MerkleRoot        string // little endian hex format
	//time is not used, eventually need to deprecate
	Time       string // little endian hex format
	Difficulty string // little endian hex format
}

//expects a string of the form `"Version": "001"`... etc to parse as a JSON
//all message fields being provided to block header must be little endian since that is what the hashing function expects
func ConvertToBlockHeader(message string) BlockHeader {
	// string will look like a JSON object
	// first convert the string into a map
	// return a BlockHeader object using map values
	var bi map[string]string             //create an empty map to put string variables into
	json.Unmarshal([]byte(message), &bi) //unmarshal string and put into bi (block info) map
	return BlockHeader{
		Version:           bi["Version"],           //little endian
		PreviousBlockHash: bi["PreviousBlockHash"], //little endian
		MerkleRoot:        bi["MerkleRoot"],        //little endian
		Time:              bi["Time"],              //little endian
		Difficulty:        bi["Difficulty"],        //little endian
	}

}

//converts a block header to a string, used to create a Message which the validation instance can pass to the msg bus
func ConvertBlockHeaderToString(h BlockHeader) string {
	return fmt.Sprintf(`{\"Version\":\"%s\",\"PreviousBlockHash\":\"%s\",\"MerkleRoot\":\"%s\",\"Time\":\"%s\",\"Difficulty\":\"%s\"}`, h.Version, h.PreviousBlockHash, h.MerkleRoot, h.Time, h.Difficulty)
}

//convert the string of a uint64 into a little endian hex format
func uintToLittleEndian(x string) (string, error) {
	u, err := strconv.ParseUint(x, 10, 64) //convert string to uint64
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, u)
	return fmt.Sprintf("%x", buf[:4]), err
}

//convert a big endian hex to a little endian hex
func reverseHexNumber(x string) [32]byte {
	newNum := ""
	for i := 0; i < len(x); i = i + 2 {
		newNum = x[i:i+2] + newNum
	}
	//pass newNum to NewHashFromString
	res := NewHashFromStr(newNum)
	return res
}

func NewHashFromStr(hash string) [32]byte {
	ret := new(chainhashOnline.Hash)
	chainhashOnline.Decode(ret, hash)
	res := [32]byte{}
	for i := 0; i < len(ret); i++ {
		res[i] = ret[i]
	}
	return res
}

// takes a given nonce and timestamp, and returns the little-endian block hash
func (bh *BlockHeader) HashInput(nonce string, timestamp string) [32]byte {
	sVersion, _ := strconv.ParseInt(bh.Version, 16, 32)
	sTime, _ := strconv.ParseInt(timestamp, 16, 32)
	sDifficulty, _ := strconv.ParseInt(bh.Difficulty, 16, 32)
	sNonce, _ := strconv.Atoi(nonce)

	//PrevBlock and MerkleRoot need to be little-endian
	newBlockHash := wire.BlockHeader{
		Version:    int32(sVersion),
		PrevBlock:  NewHashFromStr(bh.PreviousBlockHash),
		MerkleRoot: NewHashFromStr(bh.MerkleRoot),
		Timestamp:  time.Unix(sTime, 0),
		Bits:       uint32(sDifficulty),
		Nonce:      uint32(sNonce),
	}
	hash := newBlockHash.BlockHash() //little-endian
	return hash

}

func (bh *BlockHeader) UpdateHeaderInformation(_version string, _previousBlockHash string, _merkleRoot string, _time string, _difficulty string) {
	bh.Version = _version
	bh.PreviousBlockHash = _previousBlockHash
	bh.MerkleRoot = _merkleRoot
	bh.Time = _time
	bh.Difficulty = _difficulty
}

//converts a block hash to a big integer. returns an error if conversion fails
func BlockHashToBigInt(hash [32]byte) (*big.Int, error) {
	//convert input to chainhash.Hash
	chash, err := chainhashOnline.NewHash(hash[:])
	return blockchain.HashToBig(chash), err
}

//expects the block difficulty as a uint32, returns a big int
func DifficultyToBigInt(diff uint32) *big.Int {
	return blockchain.CompactToBig(diff)
	//return blockchain.CalcWork(diff)
}
