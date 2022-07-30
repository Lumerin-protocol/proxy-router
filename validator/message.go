package validator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

//import "strconv"
//import "errors"

type StratumSubmitMessage struct {
	Params []string `json:"params"`
	Id     string   `json:"id"`
	Method string   `json:"method"`
}

type StratumNotifyMessage struct {
	Id      uint     `json:"id"`
	Jsonrpc string   `json:"jsonrpc"`
	Method  string   `json:"method"`
	Params  []string `json:"params"`
}

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
func ReceiveHashingRequest(m string) (HashingInstance, error) {
	res := HashingInstance{}
	err := json.Unmarshal([]byte(m), &res)
	if err != nil {
		return res, fmt.Errorf("error receiving hashing request: %s", err)
	}
	return res, nil
}

//request to compare the given hash with the calculated hash given the nonce and timestamp compared
//to the current block
func ReceiveHashResult(m string) (HashResult, error) {
	res := HashResult{}
	err := json.Unmarshal([]byte(m), &res)
	if err != nil {
		return res, fmt.Errorf("error returning a hash result: %s", err)
	}
	return res, nil
}

//receives the number of hashes that have been counted
func ReceiveHashCount(m string) (HashCount, error) {
	res := HashCount{}
	err := json.Unmarshal([]byte(m), &res)
	if err != nil {
		return res, fmt.Errorf("error unmarshaling ReceiveHashCount: %s", err)
	}
	return res, nil
}

//request to make a new validation object
func ReceiveNewValidatorRequest(m string) (NewValidator, error) {
	res := NewValidator{}
	err := json.Unmarshal([]byte(m), &res)
	if err != nil {
		return res, fmt.Errorf("error unmarshaling ReceiveHashCount: %s", err)
	}
	return res, nil
}

//message requesting info from the validator. Validator returns everything
//and its up to the recipient to figure out what it is looking for
func ReceiveValidatorInfoRequest(m string) (GetValidationInfo, error) {
	res := GetValidationInfo{}
	err := json.Unmarshal([]byte(m), &res)
	if err != nil {
		return res, fmt.Errorf("error unmarshaling ReceiveValidatorInfoRequest: %s", err)
	}
	return res, nil
}

//message for when a new blockheader is updated
func ReceiveHeaderUpdateRequest(m string) (UpdateBlockHeader, error) {
	res := UpdateBlockHeader{}
	err := json.Unmarshal([]byte(m), &res)
	if err != nil {
		return res, fmt.Errorf("error unmarshaling ReceiveHeaderUpdateRequerst: %s", err)
	}
	return res, nil
}

func convertJSONToSubmit(b []byte) (MiningSubmit, error) {

	var msg = StratumSubmitMessage{}
	var submit = MiningSubmit{}
	json.Unmarshal(b, &msg)

	if msg.Method != "mining.submit" {
		return submit, fmt.Errorf("not a mining submit message: %+v", msg)
	}

	if len(msg.Params) != 5 {
		return submit, fmt.Errorf("paramlist length: %+v", len(msg.Params))
	}
	//WorkerName, JobID, ExtraNonce2, NTime, NOnce string

	submit.WorkerName = msg.Params[0]
	submit.JobID = msg.Params[1]
	submit.ExtraNonce2 = msg.Params[2]
	submit.NTime = msg.Params[3]
	submit.NOnce = msg.Params[4]

	return submit, nil
}

func convertJSONToNotify(b []byte) (MiningNotify, error) {

	var msg = StratumNotifyMessage{}
	var notify = MiningNotify{}
	json.Unmarshal(b, &msg)
	if msg.Method != "mining.notify" {
		return notify, fmt.Errorf("not a mining notify message: %+v", msg.Method)
	}

	if len(msg.Params) != 9 {
		return notify, fmt.Errorf("paramlist length: %+v", len(msg.Params))
	}

	notify.JobID = msg.Params[0]
	notify.PreviousBlockHash = msg.Params[1]
	notify.GTP1 = msg.Params[2]
	notify.GTP2 = msg.Params[3]
	notify.MerkleList = msg.Params[4]
	notify.Version = msg.Params[5]
	notify.NBits = msg.Params[6]
	notify.NTime = msg.Params[7]
	//default to false but need to figure out how to extract from param list
	notify.CleanJobs = false
	return notify, nil
}
