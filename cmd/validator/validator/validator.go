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

	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

const EMA_INTERVAL = 600

type diffEMA struct {
	diff		int
	lastCalc	time.Time
}

//creates a channel object which can be used to access created validators
type MainValidator struct {
	channel    Channels
	Ps         *msgbus.PubSub
	Ctx        context.Context
	MinerDiffs lumerinlib.ConcurrentMap // current difficulty target for each miner
	MinersVal  lumerinlib.ConcurrentMap // miners with a validation channel open for them
	newDiff	   chan int
}

//creates a validator
//func createValidator(bh blockHeader.BlockHeader, hashRate uint, limit uint, diff uint, messages chan message.Message) error{
func (v *MainValidator) createValidator(minerId msgbus.MinerID, bh BlockHeader, hashRate uint, limit uint, diff uint, pc string, messages chan Message) {
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
			msgbus.MinerID(m.Address),
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

//creates a new validator which can spawn multiple validation instances
func MakeNewValidator(Ctx *context.Context) *MainValidator {
	ch := Channels{
		ValidationChannels: make(map[string]chan Message),
	}
	ctxStruct := contextlib.GetContextStruct(*Ctx)
	validator := MainValidator{
		channel: ch,
		Ps:      ctxStruct.MsgBus,
		Ctx:     *Ctx,
	}
	validator.MinerDiffs.M = make(map[string]interface{})
	validator.MinersVal.M = make(map[string]interface{})
	validator.newDiff = make(chan int)
	return &validator
}

func (v *MainValidator) Start() error {
	contextlib.Logf(v.Ctx, log.LevelInfo, "Validator Starting")

	// Monitor Validation Stratum Messages
	validateEventChan := msgbus.NewEventChan()
	_, err := v.Ps.Sub(msgbus.ValidateMsg, "", validateEventChan)
	if err != nil {
		contextlib.Logf(v.Ctx, log.LevelError, "Failed to subscribe to validate events, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
		return err
	}
	go v.validateHandler(validateEventChan)

	// Monitor Miner Publish/Unpublish Events
	minersEventChan := msgbus.NewEventChan()
	_, err = v.Ps.Sub(msgbus.MinerMsg, "", minersEventChan)
	if err != nil {
		contextlib.Logf(v.Ctx, log.LevelError, "Failed to subscribe to all miner events, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
		return err
	}
	go v.minersHandler(minersEventChan)

	// Monitor Miner Update Events for all miners 
	minerIds, err := v.Ps.MinerGetAllWait()
	if err != nil {
		contextlib.Logf(v.Ctx, log.LevelError, "Failed to get all miners, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
		return err
	}
	for _, minerId := range minerIds {
		minerEventChan := msgbus.NewEventChan()
		_, err = v.Ps.Sub(msgbus.MinerMsg, msgbus.IDString(minerId), minerEventChan)
		if err != nil {
			contextlib.Logf(v.Ctx, log.LevelError, "Failed to subscribe to all miner events, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
			return err
		}
		go v.minerHandler(minerId, minerEventChan)
	}
	return nil
}

func (v *MainValidator) minersHandler(ch msgbus.EventChan) {
	for {
		select {
		case <-v.Ctx.Done():
			contextlib.Logf(v.Ctx, log.LevelInfo, "Cancelling current validator context: cancelling minersHandler go routine")
			return

		case event := <-ch:
			id := msgbus.MinerID(event.ID)

			switch event.EventType {
			case msgbus.PublishEvent:
				minerEventChan := msgbus.NewEventChan()
				_, err := v.Ps.Sub(msgbus.MinerMsg, msgbus.IDString(id), minerEventChan)
				if err != nil {
					contextlib.Logf(v.Ctx, log.LevelPanic, "Failed to subscribe to miner events, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
				}
				go v.minerHandler(id, minerEventChan)
				
			case msgbus.UnpublishEvent:
				contextlib.Logf(v.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Got Miner Unpublish Event: %v", event)
		
				contextlib.Logf(v.Ctx, log.LevelInfo, "Closing validator instance for Miner: %v", id)
				v.MinersVal.Delete(string(id))
				var closeMessage = Message{}
				closeMessage.Address = string(id)
				closeMessage.MessageType = "closeValidator"
				v.SendMessageToValidator(closeMessage)
			}
		}
	}
}

func (v *MainValidator) minerHandler(minerId msgbus.MinerID, ch msgbus.EventChan) {
	for {
		select {
		case <-v.Ctx.Done():
			contextlib.Logf(v.Ctx, log.LevelInfo, "Cancelling current validator context: cancelling minerHandler go routine")
			return

		case event := <-ch:
			switch event.EventType {
			case msgbus.UpdateEvent:
				id := msgbus.MinerID(event.ID)
				if id != minerId {
					contextlib.Logf(v.Ctx, log.LevelPanic, lumerinlib.Funcname()+"Got miner event with wrong id for miner %s: %v", minerId, event)
				}

				var miner msgbus.Miner
				switch event.Data.(type) {
				case msgbus.Miner:
					miner = event.Data.(msgbus.Miner)
				case *msgbus.Miner:
					m := event.Data.(*msgbus.Miner)
					miner = *m
				}

				if miner.State == msgbus.OfflineState  {
					timeOfflineLimit := time.Minute * 10
					timeOffline := time.Since(miner.StateChange)
					if timeOffline > timeOfflineLimit && v.MinersVal.Exists(string(id)) { // close validator instance for this miner if its been offline for more than 10 minutes
						contextlib.Logf(v.Ctx, log.LevelInfo, "Closing validator instance for Miner: %v", id)
						v.MinersVal.Delete(string(id))
						var closeMessage = Message{}
						closeMessage.Address = string(id)
						closeMessage.MessageType = "closeValidator"
						v.SendMessageToValidator(closeMessage)
					}
				}

			case msgbus.UnpublishEvent:
				contextlib.Logf(v.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Got Miner Unpublish Event for miner %s: %v", minerId, event)
				return
			}
		}
	}
}

func (v *MainValidator) validateHandler(ch msgbus.EventChan) {
	for {
		select {
		case <-v.Ctx.Done():
			contextlib.Logf(v.Ctx, log.LevelInfo, "Cancelling current validator context: cancelling validateHandler go routine")
			return

		case event := <-ch:
			if event.EventType == msgbus.PublishEvent {
				validateMsg := event.Data.(*msgbus.Validate)
				minerID := msgbus.MinerID(validateMsg.MinerID)

				loop:
				switch validateMsg.Data.(type) {
				case *msgbus.SetDifficulty:
					contextlib.Logf(v.Ctx, log.LevelTrace, lumerinlib.Funcname()+" Got Set Difficulty Msg: %v", event)
					setDifficultyMsg := validateMsg.Data.(*msgbus.SetDifficulty)
					newDiff := diffEMA{
						diff: setDifficultyMsg.Diff,
					}
					if !v.MinerDiffs.Exists(string(minerID)) { // initialze ema of diff
						newDiff.lastCalc = time.Now()
						v.MinerDiffs.Set(string(minerID), newDiff)
						go v.difficultyEMA(minerID)
					} else {
						v.newDiff <- setDifficultyMsg.Diff
					}
					if !v.MinersVal.Exists(string(minerID)) { // first time seeing miner
						v.MinersVal.Set(string(minerID), false)
					}

				case *msgbus.Notify:
					contextlib.Logf(v.Ctx, log.LevelTrace, lumerinlib.Funcname()+" Got Notify Msg: %v", event)
					if !v.MinerDiffs.Exists(string(minerID)) { // did not get set diffculty message for miner yet
						break loop
					}
					notifyMsg := validateMsg.Data.(*msgbus.Notify)
					username := notifyMsg.UserID
					version := notifyMsg.Version
					previousBlockHash := notifyMsg.PrevBlockHash
					nBits := notifyMsg.Nbits
					time := notifyMsg.Ntime
					difficulty := v.MinerDiffs.Get(string(minerID)).(diffEMA)
					diffStr := strconv.Itoa(difficulty.diff) // + 0x22000000
					diffEndian, _ := uintToLittleEndian(diffStr)
					diffBigEndian := SwitchEndian(diffEndian)

					merkelBranches := notifyMsg.MerkelBranches
					merkelBranchesStr := []string{}
					for _, m := range merkelBranches {
						merkelBranchesStr = append(merkelBranchesStr, m.(string))
					}

					merkelRootStr := ""
					if len(merkelBranchesStr) != 0 {
						merkelRoot, err := ConvertMerkleBranchesToRoot(merkelBranchesStr)
						if err != nil {
							contextlib.Logf(v.Ctx, log.LevelError, "Failed to convert merkel branches to merkel root, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
							break loop	
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
							HashRate:   "",                   // not needed for now
							Limit:      "",                   // not needed for now
							Diff:       diffBigEndian,        // highest difficulty allowed using difficulty encoding
							WorkerName: username, 		  // worker name assigned to an individual mining rig. used to ensure that attempts are being allocated correctly
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

				case *msgbus.Submit:
					contextlib.Logf(v.Ctx, log.LevelTrace, lumerinlib.Funcname()+" Got Submit Msg: %v", event)
					submitMsg := validateMsg.Data.(*msgbus.Submit)
					workername := submitMsg.WorkerName
					jobID := submitMsg.JobID
					extraNonce := submitMsg.Extraonce
					nTime := submitMsg.NTime
					nonce := submitMsg.NOnce

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
					
					if v.MinersVal.Get(string(minerID)).(bool) {
						response := v.SendMessageToValidator(tabulationMessage)
						contextlib.Logf(v.Ctx, log.LevelInfo, "Response from sending tabulation msg for Miner %v, Response: %v", minerID, response)
					}
					
				default:
					contextlib.Logf(v.Ctx, log.LevelTrace, lumerinlib.Funcname()+" Got Validate Msg with different type: %v", event)
				}

				// delete validate msg from msgbus once finished with it
				v.Ps.Unpub(msgbus.ValidateMsg, msgbus.IDString(validateMsg.ID))
			}
		}
	}
}

func (v *MainValidator) difficultyEMA(minerId msgbus.MinerID) {
	contextlib.Logf(v.Ctx, log.LevelInfo, "Starting Difficulty EMA routine for Miner: %v", minerId)
	// Monitor Miner Unpublish Events
	minerEventChan := msgbus.NewEventChan()
	_, err := v.Ps.Sub(msgbus.MinerMsg, msgbus.IDString(minerId), minerEventChan)
	if err != nil {
		contextlib.Logf(v.Ctx, log.LevelError, "Failed to subscribe to miner events, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
		return
	}
	for {
		select {
		case event := <- minerEventChan:
			if event.EventType == msgbus.UnpublishEvent {
				contextlib.Logf(v.Ctx, log.LevelInfo, lumerinlib.Funcname()+"Miner unpublished: cancelling hashrate calculator routines for Miner: %v", minerId)
				contextlib.Logf(v.Ctx, log.LevelInfo, "Closing Difficulty EMA routine for Miner: %v", minerId)
				
				id := msgbus.MinerID(event.ID)
				v.MinerDiffs.Delete(string(id))
			}
		case diff := <- v.newDiff:
			if !v.MinerDiffs.Exists(string(minerId)) {
				return
			}
			currDiff := v.MinerDiffs.Get(string(minerId)).(diffEMA)

			timePassed := time.Now().Sub(currDiff.lastCalc).Seconds()
			timeRatio := timePassed/EMA_INTERVAL

			alpha := 1 - 1.0/math.Exp(timeRatio)
			r := int(alpha*float64(diff) + (1 - alpha)*float64(currDiff.diff))
			currDiff.diff = r
			currDiff.lastCalc = time.Now()

			v.MinerDiffs.Set(string(minerId), currDiff)
		}
	}
}

func (v *MainValidator) hashrateCalculator(instance *Validator, minerId msgbus.MinerID) {
	contextlib.Logf(v.Ctx, log.LevelInfo, "Starting Hashrate Calculator routine for Miner: %v", minerId)

	for {
		if !v.Ps.MinerExistsWait(minerId) {
			return // miner unpublished
		} 
		miner, err := v.Ps.MinerGetWait(minerId)
		if err != nil {
			contextlib.Logf(v.Ctx, log.LevelError, "Failed to get miner, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
			return
		}

		// calculate 5 minute moving average of hashrate
		startHashCount := instance.HashesAnalyzed
		timeInterval := time.Second * EMA_INTERVAL
		time.Sleep(timeInterval)
		endHashCount := instance.HashesAnalyzed
		hashesAnalyzed := endHashCount - startHashCount
		if !v.MinerDiffs.Exists(string(minerId)) {
			contextlib.Logf(v.Ctx, log.LevelInfo, "Closing Hashrate Calculator routine for Miner: %v", minerId)
			return
		}
		poolDifficulty := v.MinerDiffs.Get(string(minerId)).(diffEMA)

		contextlib.Logf(v.Ctx, log.LevelTrace, lumerinlib.Funcname()+" Current Pool Difficulty: %d", poolDifficulty.diff)
		contextlib.Logf(v.Ctx, log.LevelTrace, lumerinlib.Funcname()+" Current Hashes Analyzed in this interval: %d", hashesAnalyzed)

		//calculate the number of hashes represented by the pool difficulty target
		bigDiffTarget := big.NewInt(int64(poolDifficulty.diff))
		bigHashesAnalyzed := big.NewInt(int64(hashesAnalyzed))

		result := new(big.Int).Exp(big.NewInt(2), big.NewInt(32), nil)
		hashesPerSubmission := new(big.Int).Mul(bigDiffTarget, result)
		totalHashes := new(big.Int).Mul(hashesPerSubmission, bigHashesAnalyzed)

		//divide represented hashes by time duration
		rateBigInt := new(big.Int).Div(totalHashes, big.NewInt(int64(timeInterval.Seconds())))
		hashrate := int(rateBigInt.Int64())

		// take hourly average of hashrate
		if len(instance.Hashrates) >= 6 {
			instance.Hashrates = instance.Hashrates[1:]
		} 
		hashSum := 0
		for _,h := range instance.Hashrates {
			hashSum += h
		}
		hashSum += hashrate
		newHashrate := hashSum/(len(instance.Hashrates) + 1)
		instance.Hashrates = append(instance.Hashrates, newHashrate)
		
		contextlib.Logf(v.Ctx, log.LevelTrace, lumerinlib.Funcname()+" Current Hashrate Moving Average for Miner %s: %d", miner.ID, newHashrate)

		// update miner with new hashrate value and fix slicing percentages accordingly
		miner, err = v.Ps.MinerGetWait(minerId)
		if err != nil {
			contextlib.Logf(v.Ctx, log.LevelError, "Failed to get miner, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
			return
		}
		miner.CurrentHashRate = newHashrate
		timeSlice := false
		if len(miner.Contracts) > 0 && len(instance.Hashrates) > 1 {
			hashrateUpdateFactor := float64(instance.Hashrates[len(instance.Hashrates) - 2])/float64(newHashrate)
			for i, v := range miner.Contracts {
				newSliceFactor := v*hashrateUpdateFactor
				if v < 1 && newSliceFactor < 1 {
					miner.Contracts[i] = newSliceFactor
				} else if v < 1 && newSliceFactor >= 1 {
					miner.Contracts[i] = 1
				}

				// check if miner still needs to be sliced after updates
				if miner.Contracts[i] < 1 {
					timeSlice = true
				}
			}
		}
		if timeSlice {
			miner.TimeSlice = true
		}
		v.Ps.MinerSetWait(*miner)
	}
}