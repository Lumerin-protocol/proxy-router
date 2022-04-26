package message

import "encoding/json"
import "reflect"
import "fmt"
import "strings"

//JSON can desearealize to have these values
//base message that will be decoded to first to determine where message should go
type Message struct {
	//address is an ethereum address
	//MessageType is a string which describes which message is being sent
	//Message is a stringified JSON message
	Address, MessageType, Message string
}

//struct for new validator message
type NewValidator struct {
	//BH is a block header
	//HashRate is the smart contract defined hashrate
	//Limit is the number of hashes that the contract has promised to deliver
	//Diff is pool difficulty target which submitted hashes must fall under
	BH, HashRate, Limit, Diff, WorkerName string
}

//struct for hashing message
//rebuilding to mirror content of mining.submit
//Username and JobID are not used. They're included to ease the process of deserializing the
//mining.submit message into a HashingInstance struct
type HashingInstance struct {
	WorkerName, JobID, ExtraNonce2, NTime, NOnce string
}

//struct for requesting information from validator
type GetValidationInfo struct {
	Hashes, Duration string
}

//struct to update the block header information within the validator
type UpdateBlockHeader struct {
	Version, PreviousBlockHash, MerkleRoot, Time, Difficulty string
}

type HashResult struct { //string will be true or false
	IsCorrect string
}

type HashCount struct { //string will be an integer
	HashCount string
}

//used for the mining rig hashrate tracking process
type TabulationCount struct { //string will be an integer
	HashCount uint
}

//for testing purposes for now
type MiningNotify struct {
	JobID, PreviousBlockHash, GTP1, GTP2, MerkleList, Version, NBits, NTime string
	CleanJobs                                                               bool
}

//contains all the field of a stratum mining.submit message
type MiningSubmit struct {
	WorkerName, JobID, ExtraNonce2, NTime, NOnce string
}

func ConvertMessageToString(i interface{}) string {
	v := reflect.ValueOf(i)
	myString := "{"
	for j := 0; j < v.NumField(); j++ {
		var tempString []string
		newString := fmt.Sprintf(`"%s":"%s"`, v.Type().Field(j).Name, v.Field(j).Interface())
		tempString = []string{myString, newString}
		if myString == "{" {
			myString = strings.Join(tempString, "")
		} else {
			myString = strings.Join(tempString, ",")
		}
	}
	myString += "}"
	return myString
}

//request to compare the given hash with the calculated hash given the nonce and timestamp compared
//to the current block
func ReceiveHashingRequest(m string) HashingInstance {
	res := HashingInstance{}
	json.Unmarshal([]byte(m), &res)
	return res
}

//request to compare the given hash with the calculated hash given the nonce and timestamp compared
//to the current block
func ReceiveHashResult(m string) HashResult {
	res := HashResult{}
	json.Unmarshal([]byte(m), &res)
	return res
}

//receives the number of hashes that have been counted
func ReceiveHashCount(m string) HashCount {
	res := HashCount{}
	json.Unmarshal([]byte(m), &res)
	return res
}

//request to make a new validation object
func ReceiveNewValidatorRequest(m string) NewValidator {
	res := NewValidator{}
	json.Unmarshal([]byte(m), &res)
	return res
}

//message requesting info from the validator. Validator returns everything
//and its up to the recipient to figure out what it is looking for
func ReceiveValidatorInfoRequest(m string) GetValidationInfo {
	res := GetValidationInfo{}
	json.Unmarshal([]byte(m), &res)
	return res
}

//message for when a new blockheader is updated
func ReceiveHeaderUpdateRequest(m string) UpdateBlockHeader {
	res := UpdateBlockHeader{}
	json.Unmarshal([]byte(m), &res)
	return res
}
