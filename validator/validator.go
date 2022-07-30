/*
this is the main package where a goroutine is spun off to be the validator
incoming messages are a JSON object with the following key-value pairs:
	messageType: string
	contractAddress: string
	message: string

	messageType is the type of message, one of the following: "create", "validate", "getHashRate", "updateBlockHeader" [more]
	contractAddress will always be a single ethereum address
*/

package validator

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lumerinlib"
	"gitlab.com/TitanInd/hashrouter/miner"
)

const EMA_INTERVAL = 600

type diffEMA struct {
	diff     int
	lastCalc time.Time
}

//creates a channel object which can be used to access created validators
type MainValidator struct {
	channel    Channels
	MinerDiffs lumerinlib.ConcurrentMap // current difficulty target for each miner
	MinersVal  lumerinlib.ConcurrentMap // miners with a validation channel open for them
	newDiff    chan int
	log        interfaces.ILogger
	minerRepo  *miner.MinerRepo
}

//creates a new validator which can spawn multiple validation instances
func MakeValidator(Ctx *context.Context) *MainValidator {
	ch := Channels{
		ValidationChannels: make(map[string]chan Message),
	}
	validator := MainValidator{
		channel: ch,
	}
	validator.MinerDiffs.M = make(map[string]interface{})
	validator.MinersVal.M = make(map[string]interface{})
	validator.newDiff = make(chan int)
	return &validator
}

func (v *MainValidator) OnMinerUnpublish(ctx context.Context, id string) {
	v.log.Debugf("Got Miner Unpublish/Unsubscribe Event: %v", id)
	v.MinersVal.Delete(string(id))
	v.MinerDiffs.Delete(string(id))
}

func (v *MainValidator) OnMinerSetDifficulty(ctx context.Context, minerID string, diff int) {
	v.log.Debugf("Got Set Difficulty Msg: %s %v", minerID, diff)
	newDiff := diffEMA{
		diff: diff,
	}
	if !v.MinerDiffs.Exists(string(minerID)) { // initialze ema of diff
		newDiff.lastCalc = time.Now()
		v.MinerDiffs.Set(string(minerID), newDiff)
		v.onNewDiff(minerID, diff)
	} else {
		v.onNewDiff(minerID, diff)
	}
	if !v.MinersVal.Exists(string(minerID)) { // first time seeing miner
		v.MinersVal.Set(string(minerID), false)
	}
}

func (v *MainValidator) OnMinerNotify(ctx context.Context, minerID string, username string, version string, previousBlockHash string, nBits string, time string, merkelBranches []string) {
	v.log.Debugf("Got Notify Msg: %v", minerID)
	if !v.MinerDiffs.Exists(string(minerID)) { // did not get set diffculty message for miner yet
		return
	}

	difficulty := v.MinerDiffs.Get(string(minerID)).(diffEMA)
	diffStr := strconv.Itoa(difficulty.diff) // + 0x22000000
	diffEndian, _ := uintToLittleEndian(diffStr)
	diffBigEndian := SwitchEndian(diffEndian)

	merkelBranchesStr := []string{}
	for _, m := range merkelBranches {
		merkelBranchesStr = append(merkelBranchesStr, m)
	}

	merkelRootStr := ""
	if len(merkelBranchesStr) != 0 {
		merkelRoot, err := ConvertMerkleBranchesToRoot(merkelBranchesStr)
		if err != nil {
			v.log.Panicf("Failed to convert merkel branches to merkel root %w", err)
		}
		merkelRootStr = merkelRoot.String()
	}

	blockHeader := ConvertBlockHeaderToString(BlockHeader{
		Version:           version,
		PreviousBlockHash: previousBlockHash,
		MerkleRoot:        merkelRootStr,
		Time:              time,
		Difficulty:        nBits,
	})

	if !v.MinersVal.Get(string(minerID)).(bool) { // no validation channel for miner yet
		var createMessage = Message{}
		createMessage.Address = string(minerID)
		createMessage.MessageType = "createNew"
		createMessage.Message = ConvertMessageToString(NewValidator{
			BH:         blockHeader,
			HashRate:   "",            // not needed for now
			Limit:      "",            // not needed for now
			Diff:       diffBigEndian, // highest difficulty allowed using difficulty encoding
			WorkerName: username,      // worker name assigned to an individual mining rig. used to ensure that attempts are being allocated correctly
		})
		v.SendMessageToValidator(createMessage)
		v.MinersVal.Set(string(minerID), true)
	} else { // update block header in existing validation channel
		var updateMessage = Message{}
		updateMessage.Address = string(minerID)
		updateMessage.MessageType = "blockHeaderUpdate"
		updateMessage.Message = ConvertMessageToString(UpdateBlockHeader{
			Version:           version,
			PreviousBlockHash: previousBlockHash,
			MerkleRoot:        merkelRootStr,
			Time:              time,
			Difficulty:        nBits,
		})
		v.SendMessageToValidator(updateMessage)
	}
}

func (v *MainValidator) OnMinerSubmit(ctx context.Context, minerID, workername, jobID, extraNonce, nTime, nonce string) {
	v.log.Debugf("Got Submit Msg")

	var tabulationMessage = Message{}
	mySubmit := MiningSubmit{}
	mySubmit.WorkerName = workername
	mySubmit.JobID = jobID
	mySubmit.ExtraNonce2 = extraNonce
	mySubmit.NTime = nTime
	mySubmit.NOnce = nonce
	tabulationMessage.Address = string(minerID)
	tabulationMessage.MessageType = "tabulate"
	tabulationMessage.Message = ConvertMessageToString(mySubmit)

	v.SendMessageToValidator(tabulationMessage)
}

//creates a validator
//func createValidator(bh blockHeader.BlockHeader, hashRate uint, limit uint, diff uint, messages chan message.Message) error{
func (v *MainValidator) createValidator(minerId string, bh BlockHeader, hashRate uint, limit uint, diff uint, pc string, messages chan Message) {
	go func() {
		myValidator := Validator{
			BH:               bh,
			StartTime:        time.Now(),
			HashesAnalyzed:   0,
			DifficultyTarget: diff,
			ContractHashRate: hashRate,
			ContractLimit:    limit,
			PoolCredentials:  pc, // pool login credentials
		}
		go v.hashrateCalculator(&myValidator, minerId)
		for {
			//message is of type message, with messageType and content values
			m := <-messages
			if m.MessageType == "validate" {
				//potentially bubble up result of function call
				req, hashingRequestError := ReceiveHashingRequest(m.Message)
				if hashingRequestError != nil {
					//error handling for hashing request error
				}
				result, hashingErr := myValidator.IncomingHash(req.WorkerName, req.NOnce, req.NTime) //this function broadcasts a message
				newM := m
				if hashingErr != "" { //make this error the message contents precedded by ERROR
					newM.Message = fmt.Sprintf("ERROR: error encountered validating a mining.submit message: %s\n", hashingErr)
				} else {
					newM.Message = ConvertMessageToString(result)
				}
				messages <- newM //sends the message.HashResult struct into the channel
			} else if m.MessageType == "getHashCompleted" {
				//print number of hashes done
				result := HashCount{}
				result.HashCount = strconv.FormatUint(uint64(myValidator.HashesAnalyzed), 10)
				newM := m
				newM.Message = ConvertMessageToString(result)
				messages <- newM
				//create a response object where the result is the hashes analyzed

			} else if m.MessageType == "blockHeaderUpdate" {
				bh := ConvertToBlockHeader(m.Message)
				myValidator.UpdateBlockHeader(bh)
				newM := m
				messages <- newM
			} else if m.MessageType == "closeValidator" {
				close(messages)
				return
			} else if m.MessageType == "tabulate" {
				/*
					this is similar to the validation message, but instead of returning a boolean value, it returns the current hashrate after the message is sent to it
				*/
				result := TabulationCount{}
				req, hashingRequestError := ReceiveHashingRequest(m.Message)
				if hashingRequestError != nil {
					//error handling for hashing request error
				}
				myValidator.IncomingHash(req.WorkerName, req.NOnce, req.NTime) //this function broadcasts a message
				hashrate := myValidator.UpdateHashrate()
				result.HashCount = hashrate
				newM := m
				newM.Message = ConvertMessageToString(result)
				messages <- newM

			}
		}
	}()
}

//entry point of all validators
//rite now it only returns whether or not a hash was successful. Future abilities should be able to return a response based on the input message
func (v *MainValidator) SendMessageToValidator(m Message) *Message {
	if m.MessageType == "createNew" {
		newChannel := v.channel.AddChannel(m.Address)
		//need to extract the block header out of m.Message
		creation, creationErr := ReceiveNewValidatorRequest(m.Message)
		if creationErr != nil {
			//error handling for validator creation
		}
		useDiff, _ := strconv.ParseUint(creation.Diff, 16, 32)
		//fmt.Println("useDiff:",useDiff)
		v.createValidator( //creation["BH"] is an embedded JSON object
			m.Address,
			ConvertToBlockHeader(creation.BH),
			ConvertStringToUint(creation.HashRate),
			ConvertStringToUint(creation.Limit),
			uint(useDiff),
			creation.WorkerName,
			newChannel,
		)
		return nil
	} else { //any other message will be sent to the validator, where the internal channel logic will handle the message
		channel, _ := v.channel.GetChannel(m.Address)
		channel <- m
		returnMessageMessage := <-channel
		//returnMessageMessage is a message of type message.HashResult
		var returnMessage = Message{}
		returnMessage.Address = m.Address
		returnMessage.MessageType = "response"
		returnMessage.Message = returnMessageMessage.Message
		return &returnMessage
	}
}

func (v *MainValidator) ReceiveJSONMessage(b []byte, id string) {

	//blindly try to convert the message to a submit message. If it returns true
	//process the message
	msg := Message{}
	msg.Address = id
	submit, err := convertJSONToSubmit(b)
	//we don't care about the error message
	if err == nil {
		msg.MessageType = "validate"
		msg.Message = ConvertMessageToString(submit)
	}

	//blindly try to convert the message to a notify message.
	notify, err := convertJSONToNotify(b)
	if err == nil {
		msg.MessageType = "blockHeaderUpdate"
		msg.Message = ConvertMessageToString(notify)
	}
	//send message to validator.
	v.SendMessageToValidator(msg)

}

func (v *MainValidator) onNewDiff(minerId string, diff int) {
	if !v.MinerDiffs.Exists(string(minerId)) {
		return
	}
	currDiff := v.MinerDiffs.Get(string(minerId)).(diffEMA)

	timePassed := time.Now().Sub(currDiff.lastCalc).Seconds()
	timeRatio := timePassed / EMA_INTERVAL

	alpha := 1 - 1.0/math.Exp(timeRatio)
	r := int(alpha*float64(diff) + (1-alpha)*float64(currDiff.diff))
	currDiff.diff = r
	currDiff.lastCalc = time.Now()

	v.MinerDiffs.Set(string(minerId), currDiff)
}

func (v *MainValidator) hashrateCalculator(instance *Validator, minerId string) {
	for {
		miner, ok := v.minerRepo.Load(minerId)
		if !ok {
			return // miner unpublished
		}

		// calculate 5 minute moving average of hashrate
		startHashCount := instance.HashesAnalyzed
		timeInterval := time.Second * EMA_INTERVAL
		time.Sleep(timeInterval)
		endHashCount := instance.HashesAnalyzed
		hashesAnalyzed := endHashCount - startHashCount
		if !v.MinerDiffs.Exists(string(minerId)) {
			return
		}
		poolDifficulty := v.MinerDiffs.Get(string(minerId)).(diffEMA)
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
		if len(instance.Hashrates) >= 6 {
			instance.Hashrates = instance.Hashrates[1:]
		}
		hashSum := 0
		for _, h := range instance.Hashrates {
			hashSum += h
		}
		hashSum += hashrate
		newHashrate := hashSum / (len(instance.Hashrates) + 1)
		instance.Hashrates = append(instance.Hashrates, newHashrate)

		v.log.Debugf("current Hashrate Moving Average for Miner %s: %d", miner.GetID(), newHashrate)
		// update miner with new hashrate value

		// TODO: inject this as dependency on the miner model
		// miner.CurrentHashRate = newHashrate
		// v.Ps.MinerSetWait(*miner)
	}
}
