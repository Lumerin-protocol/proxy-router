package validator

import "testing"
import "example.com/message"
import "example.com/blockHeader"
import "strings"

/*
feature tests of the validator system.
All validators start off witt the metadata of Bitcoin block 300000
*/

//function to generate a new validator message
func createTestValidator() message.Message {
	var returnMessage = message.Message{}
	returnMessage.Address = "123"
	returnMessage.MessageType = "createNew"

	//need function to convert message to string and string to message
	newValidatorMessage := message.NewValidator{
		BH:         blockHeader.ConvertBlockHeaderToString(createTestBlockHeader()),
		HashRate:   "10",        //arbitrary number
		Limit:      "10",        //arbitrary number
		Diff:       "1d00ffff",  //highest difficulty allowed using difficulty encoding
		WorkerName: "prod.s9x8", //worker name assigned to an individual mining rig. used to ensure that attempts are being allocated correctly
	}
	newValidatorString := message.ConvertMessageToString(newValidatorMessage)
	returnMessage.Message = newValidatorString
	return returnMessage
}

//function to create a new blockheader for use in a new validator.
//using UpdateBlockHeader type since it contains all necessary information
//this is constructed using Bitcoin block 300000
/*
block 300000 data obtained from a raspibolt lnd node using unix command bitcoin-cli getBlockInfo $(bitcoin-cli getBlockHash 300000)
hashes are represented in big-endian format but converted to little-endian format for testing reasons
"hash": "000000000000000082ccf8f1557c5d40b21edabb18d2d691cfbf87118bac7254",
"height": 300000,
"versionHex": "00000002",
"merkleroot": "915c887a2d9ec3f566a648bedcf4ed30d0988e22268cfe43ab5b0cf8638999d3",
"tx": [
],
"time": 1399703554,
"nonce": 222771801,
"bits": "1900896c",
"previousblockhash": "000000000000000067ecc744b5ae34eebbde14d21ca4db51652e4d67e155f07e",
*/
func createTestBlockHeader() blockHeader.BlockHeader {
	return blockHeader.BlockHeader{
		Version:           "00000002",                                                         //bitcoin difficulty big endian
		PreviousBlockHash: "000000000000000067ecc744b5ae34eebbde14d21ca4db51652e4d67e155f07e", //big-endian expected
		MerkleRoot:        "915c887a2d9ec3f566a648bedcf4ed30d0988e22268cfe43ab5b0cf8638999d3", //big-endian expected
		Time:              "1399703554",                                                       //timestamp, not necessay and overwritten with a submission attempt
		Difficulty:        "1900896c",                                                         //big-endian the difficulty target that a block needs to meet
	}
}

//this returns an updated block header due to the block being attempted
//having been mined. output should be the same as createTestBloclHeader
/*
block 300001 info for update messages
  "hash": "000000000000000049a0914d83df36982c77ac1f65ade6a52bdced2ce312aba9",
  "height": 300001,
  "versionHex": "00000002",
  "merkleroot": "7cbbf3148fe2407123ae248c2de7428fa067066baee245159bf4a37c37aa0aab",
  "time": 1399704683,
  "nonce": 3476871405,
  "bits": "1900896c",
  "previousblockhash": "000000000000000082ccf8f1557c5d40b21edabb18d2d691cfbf87118bac7254",
*/
func createBlockHeaderUpdate() blockHeader.BlockHeader {
	//this information needs to be changed. currently it is just a copy of the information in CreateTestBlockHeader
	return blockHeader.BlockHeader{
		Version:           "00000002",
		PreviousBlockHash: "000000000000000082ccf8f1557c5d40b21edabb18d2d691cfbf87118bac7254",
		MerkleRoot:        "7cbbf3148fe2407123ae248c2de7428fa067066baee245159bf4a37c37aa0aab",
		Time:              "536dcc6b",
		Difficulty:        "1900896c",
	}
}

//creates a submit message which is of the same type as stratum mining.submit
func createSubmitMessage(wn string, jid string, en2 string, nt string, no string) message.Message {
	returnMessage := message.Message{}
	mySubmit := message.MiningSubmit{}
	mySubmit.WorkerName = wn
	mySubmit.JobID = jid
	mySubmit.ExtraNonce2 = en2
	mySubmit.NTime = nt
	mySubmit.NOnce = no
	returnMessage.Address = "123"
	returnMessage.MessageType = "validate"
	returnMessage.Message = message.ConvertMessageToString(mySubmit)
	return returnMessage
}

//creates a tabulation message which is of the same type as stratum mining.submit
func createTabulationMessage(wn string, jid string, en2 string, nt string, no string, MUID string) message.Message {
	returnMessage := message.Message{}
	mySubmit := message.MiningSubmit{}
	mySubmit.WorkerName = wn
	mySubmit.JobID = jid
	mySubmit.ExtraNonce2 = en2
	mySubmit.NTime = nt
	mySubmit.NOnce = no
	returnMessage.Address = MUID //stands for miner unique ID
	returnMessage.MessageType = "tabulate"
	returnMessage.Message = message.ConvertMessageToString(mySubmit)
	return returnMessage
}

//creates a notify message which is of the same type as stratum mining.notify
func createNotifyMessage(JobID string, PreviousBlockHash string, GTP1 string, GTP2 string, MerkleList string, Version string, NBits string, NTime string, CleanJobs bool) message.Message {
	returnMessage := message.Message{}
	myNotify := message.MiningNotify{}
	myNotify.PreviousBlockHash = PreviousBlockHash
	myNotify.GTP1 = GTP1
	myNotify.GTP2 = GTP2
	myNotify.MerkleList = MerkleList
	myNotify.Version = Version
	myNotify.NBits = NBits
	myNotify.NTime = NTime
	myNotify.CleanJobs = CleanJobs
	returnMessage.Address = "123"
	returnMessage.MessageType = "validate"
	returnMessage.Message = message.ConvertMessageToString(myNotify)
	return returnMessage
}

//use time and nonce from block 300000
func createHashSubmissionMessage() message.Message {
	returnMessage := createSubmitMessage(
		"prod.s9x8", //worker name
		"d73b189a",  //job ID
		"",          //extra nonce 2
		"536dc802",  //time in bits
		"222771801") //nonce
	return returnMessage
}

func createMinerNotifyMessage() message.Message {
	returnMessage := createNotifyMessage("616c4a28", "17c2c0507d5b4f32aa1ca39d82b83f16dfbf75d000093fd60000000000000000", "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff2803cbf90a00046cf6e6610c", "0a746974616e2f6a74677261737369650affffffff024aa07e26000000001976a9143f1ad6ada38343e55cc4c332657e30a71c86b66188ac0000000000000000266a24aa21a9ed5617dd59c856a6ae1c00b6df9c4ed26727616d4d7b59f3eaacd16d810ff6dd3400000000", `[]`, "20000000", "170b8c8b", "61e6f66c", false)
	return returnMessage
}

/*
"params":["783647bc","17c2c0507d5b4f32aa1ca39d82b83f16dfbf75d000093fd60000000000000000","01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff2803cbf90a00046cf6e6610c","0a746974616e2f6a74677261737369650affffffff024aa07e26000000001976a9143f1ad6ada38343e55cc4c332657e30a71c86b66188ac0000000000000000266a24aa21a9ed5617dd59c856a6ae1c00b6df9c4ed26727616d4d7b59f3eaacd16d810ff6dd3400000000","","20000000","170b8c8b","61e6f66c",false]}

*/

//create a message which asks the validator for the current hash count
func createHashCounterRequestMessage() message.Message {
	var returnMessage = message.Message{}
	returnMessage.Address = "123"
	returnMessage.MessageType = "getHashCompleted"
	returnMessage.Message = ""
	return returnMessage
}

//creates a generic close method to terminate the specific goroutine
func createCloseMethod() message.Message {
	var returnMessage = message.Message{}
	returnMessage.Address = "123"
	returnMessage.MessageType = "closeValidator"
	returnMessage.Message = ""
	return returnMessage
}

//create a validator, send a hash, close validator
func TestCreateValidator(t *testing.T) {
	//creating a validator
	creationMessage := createTestValidator()
	validator := MakeNewValidator()
	validator.SendMessageToValidator(creationMessage)

	//sending a hash message to the validator
	hashingMessage := createHashSubmissionMessage()
	hashingResult := validator.SendMessageToValidator(hashingMessage)
	if strings.Contains(hashingResult.Message, "ERROR") { //need to update this functionality
		t.Errorf("incorrect hash: %v", hashingResult.Message)
	}

	//closing the validator
	closingResult := validator.SendMessageToValidator(createCloseMethod())
	if strings.Contains(closingResult.Message, "ERROR") { //need to update this functionality
		t.Errorf("error closing validator: %v", closingResult.Message)
	}
}

func TestHashRatePerAsic(t *testing.T) {
	/*
		this is a test to see if a mining rig can have its hashrate tabulated to within a certain degree
		steps:
		1. connect an asic to the proxy server (this is ignored and instead a new validator is created with address of 1
		2. send the same mining message to the validator every N seconds
		3. call the function to get the calculated hashrate in the validator
		4. compare to the expected hashrate
	*/
	//validator := createTestValidator()
}

//create a validator, send a hash, confirm hash results in true, close validator
func TestCreateValidatorValidateHashCloseValidator(t *testing.T) {
	//creating a validator
	creationMessage := createTestValidator()
	validator := MakeNewValidator()
	validator.SendMessageToValidator(creationMessage)

	//sending a hash message to the validator
	hashingMessage := createHashSubmissionMessage()
	hashResult := validator.SendMessageToValidator(hashingMessage)
	//should be a message where the Message is of type HashResult
	if message.ReceiveHashResult(hashResult.Message).IsCorrect != "true" {
		t.Errorf("incorrect hash: %v", message.ReceiveHashResult(hashResult.Message))
	}

	//closing the validator
	validator.SendMessageToValidator(createCloseMethod())
}

//create a validator, send a hash, confirm hash results in true, close validator
func TestSubmit2HashesVerifyCount(t *testing.T) {
	//creating a validator
	creationMessage := createTestValidator()
	validator := MakeNewValidator()
	validator.SendMessageToValidator(creationMessage)

	//sending 2 mining.submit messages to validator
	hashingMessage := createHashSubmissionMessage()
	validator.SendMessageToValidator(hashingMessage)
	validator.SendMessageToValidator(hashingMessage)

	//creating hash request message
	hashRequestMessage := createHashCounterRequestMessage()

	//obtaining the hash verify response
	//resultingHashes := validator.SendMessageToValidator(hashRequestMessage)
	hashCount := validator.SendMessageToValidator(hashRequestMessage)
	resultingHashCount := message.ReceiveHashCount(hashCount.Message).HashCount
	validator.SendMessageToValidator(createCloseMethod())
	if resultingHashCount != "2" {
		t.Errorf("incorrect hashcount: %v", resultingHashCount)
	}
}

/*
//create 2 validators, send each a hash, confirm hash results are true, close validators
func TestCreateTwoValidatorValidateHashCloseValidator(t *testing.T) {
}

//create a validator, send an invalid hash, confirm failure, close validator
func TestInvalidHashing(t *testing.T) {
}

//create 2 validators, send each a hash, close validators
func TestCreateTwoValidators(t *testing.T) {
}

//create a validator, send hashes to simulate 10 TH/s, confirm hashrate, closeout validator
func TestCaldulatedHashRate(t *testing.T) {
}

//create a validator, allow to run for 30 seconds, confirm that validator displays duration as 30 seconds
func TestTimeDurationTracking(t *testing.T) {
}

//create a validator, submit a hash, update blockheader, submit a hash, confirm that new hash conforms to new blockheader info
func TestBlockHeaderUpdate(t *testing.T) {
}
*/
