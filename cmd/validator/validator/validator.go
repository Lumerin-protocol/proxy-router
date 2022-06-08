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
	"strconv"
	"time"

	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

//creates a channel object which can be used to access created validators
type MainValidator struct {
	channel    Channels
	Ps         *msgbus.PubSub
	Ctx        context.Context
	MinerDiffs lumerinlib.ConcurrentMap // current difficulty target for each miner
	MinersVal  lumerinlib.ConcurrentMap // miners with a validation channel open for them
}

//creates a validator
//func createValidator(bh blockHeader.BlockHeader, hashRate uint, limit uint, diff uint, messages chan message.Message) error{
func createValidator(bh BlockHeader, hashRate uint, limit uint, diff uint, pc string, messages chan Message) {
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
		createValidator( //creation["BH"] is an embedded JSON object
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

	return nil
}

func (v *MainValidator) validateHandler(ch msgbus.EventChan) {
	for {
		select {
		case <-v.Ctx.Done():
			contextlib.Logf(v.Ctx, log.LevelInfo, "Cancelling current validator context: cancelling minerHandler go routine")
			return

		case event := <-ch:
			if event.EventType == msgbus.PublishEvent {
				//id := msgbus.ValidateID(event.ID)
				validateMsg := event.Data.(*msgbus.Validate)
				minerID := msgbus.MinerID(validateMsg.MinerID)
				//destID := msgbus.DestID(validateMsg.DestID)
				miner, err := v.Ps.MinerGetWait(minerID)
				if err != nil {
					contextlib.Logf(v.Ctx, log.LevelPanic, "Failed to get miner, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
				}
				// dest,err := v.Ps.DestGetWait(destID)
				// if err != nil {
				// 	contextlib.Logf(v.Ctx, log.LevelPanic, "Failed to get miner dest, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
				// }
				//workerName := dest.Username() + ":" + dest.Password()

				loop:
				switch validateMsg.Data.(type) {
				case *msgbus.SetDifficulty:
					contextlib.Logf(v.Ctx, log.LevelTrace, lumerinlib.Funcname()+" Got Set Difficulty Msg: %v", event)
					setDifficultyMsg := validateMsg.Data.(*msgbus.SetDifficulty)
					diffStr := strconv.Itoa(setDifficultyMsg.Diff + 570425344) // + 0x22000000
					diffEndian, _ := uintToLittleEndian(diffStr)
					diffBigEndian := SwitchEndian(diffEndian)
					v.MinerDiffs.Set(string(minerID), diffBigEndian)
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
					difficulty := v.MinerDiffs.Get(string(minerID)).(string)

					merkelBranches := notifyMsg.MerkelBranches
					merkelBranchesStr := []string{}
					for _, m := range merkelBranches {
						merkelBranchesStr = append(merkelBranchesStr, m.(string))
					}

					merkelRootStr := ""
					if len(merkelBranchesStr) != 0 {
						merkelRoot, err := ConvertMerkleBranchesToRoot(merkelBranchesStr)
						if err != nil {
							contextlib.Logf(v.Ctx, log.LevelPanic, "Failed to convert merkel branches to merkel root, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
						}
						merkelRootStr = merkelRoot.String()
					}
					

					// Version:           "00000002",                                                         //bitcoin difficulty big endian
					// PreviousBlockHash: "000000000000000067ecc744b5ae34eebbde14d21ca4db51652e4d67e155f07e", //big-endian expected
					// MerkleRoot:        "915c887a2d9ec3f566a648bedcf4ed30d0988e22268cfe43ab5b0cf8638999d3", //big-endian expected
					// Time:              "1399703554",                                                       //timestamp, not necessay and overwritten with a submission attempt
					// Difficulty:        "1900896c",                                                         //big-endian the difficulty target that a block needs to meet

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
							Diff:       difficulty,           // highest difficulty allowed using difficulty encoding
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

					m := v.SendMessageToValidator(tabulationMessage)
					hashCountStr,err := ReceiveHashCount(m.Message)
					if err != nil {
						contextlib.Logf(v.Ctx, log.LevelPanic, "Failed to receive hashcount msg, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
					}

					// parse hashcount field returned (need to fix how its returned e.g. {"HashCount":"%!s(uint=1537228672809129301)"}})
					hashCountRunes := []rune{}
					startFound := false
					for _,v := range m.Message {
						if v == ')' {
							break
						}
						if startFound {
							hashCountRunes = append(hashCountRunes, v)
						}
						if v == '=' {
							startFound = true
						}
					}
					hashCountStr.HashCount = string(hashCountRunes)
					
					contextlib.Logf(v.Ctx, log.LevelTrace, lumerinlib.Funcname()+" Hashrate Calculated for Miner %s: %s", miner.ID, hashCountStr.HashCount)

					hashCount, err := strconv.Atoi(hashCountStr.HashCount)
					if err != nil {
						contextlib.Logf(v.Ctx, log.LevelPanic, "Failed to convert hashrate string to int, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
					}

					// set hashrate in miner
					miner.CurrentHashRate = hashCount
					v.Ps.MinerSetWait(*miner)
				default:
					contextlib.Logf(v.Ctx, log.LevelTrace, lumerinlib.Funcname()+" Got Validate Msg with different type: %v", event)
				}
			}
		}
	}
}
