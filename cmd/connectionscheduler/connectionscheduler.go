package connectionscheduler

import (
	"context"
	"math"
	"sort"

	//"strconv"
	"sync"

	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

type ConnectionScheduler struct {
	Ps                   *msgbus.PubSub
	Contracts            lumerinlib.ConcurrentMap
	ReadyMiners          lumerinlib.ConcurrentMap // miners with no contract
	BusyMiners           lumerinlib.ConcurrentMap // miners fulfilling a contract
	NodeOperator         msgbus.NodeOperator
	MinerUpdatedChans    lumerinlib.ConcurrentMap
	ContractUpdatedChans lumerinlib.ConcurrentMap
	Passthrough          bool
	Ctx                  context.Context
}

// implement golang sort interface
type Miner struct {
	id       msgbus.MinerID
	hashrate int
}
type MinerList []Miner

func (m MinerList) Len() int           { return len(m) }
func (m MinerList) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m MinerList) Less(i, j int) bool { return m[i].hashrate < m[j].hashrate }

func New(Ctx *context.Context, NodeOperator *msgbus.NodeOperator, Passthrough bool) (cs *ConnectionScheduler, err error) {
	ctxStruct := contextlib.GetContextStruct(*Ctx)
	cs = &ConnectionScheduler{
		Ps:           ctxStruct.MsgBus,
		NodeOperator: *NodeOperator,
		Passthrough:  Passthrough,
		Ctx:          *Ctx,
	}
	cs.Contracts.M = make(map[string]interface{})
	cs.ReadyMiners.M = make(map[string]interface{})
	cs.BusyMiners.M = make(map[string]interface{})
	cs.MinerUpdatedChans.M = make(map[string]interface{})
	cs.ContractUpdatedChans.M = make(map[string]interface{})
	return cs, err
}

func (cs *ConnectionScheduler) Start() (err error) {
	contextlib.Logf(cs.Ctx, log.LevelInfo, "Connection Scheduler Starting")

	// Update connection scheduler with current contracts
	event, err := cs.Ps.GetWait(msgbus.ContractMsg, "")
	if err != nil {
		contextlib.Logf(cs.Ctx, log.LevelError, "Failed to get all contract ids, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
		return err
	}
	contracts := event.Data.(msgbus.IDIndex)
	for i := range contracts {
		event, err := cs.Ps.GetWait(msgbus.ContractMsg, contracts[i])
		if err != nil {
			contextlib.Logf(cs.Ctx, log.LevelError, "Failed to get contract, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
			return err
		}
		contract := event.Data.(msgbus.Contract)
		cs.Contracts.Set(string(contracts[i]), contract)
	}

	// Monitor New Contracts
	contractEventChan := msgbus.NewEventChan()
	_, err = cs.Ps.Sub(msgbus.ContractMsg, "", contractEventChan)
	if err != nil {
		contextlib.Logf(cs.Ctx, log.LevelError, "Failed to subscribe to contract events, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
		return err
	}
	go cs.goContractHandler(contractEventChan)

	// Update connection scheduler with current miners in online state
	miners, err := cs.Ps.MinerGetAllWait()
	if err != nil {
		contextlib.Logf(cs.Ctx, log.LevelError, "Failed to get all miner ids, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
		return err
	}
	for i := range miners {
		miner, err := cs.Ps.MinerGetWait(miners[i])
		if err != nil {
			contextlib.Logf(cs.Ctx, log.LevelError, "Failed to get miner, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
			return err
		}
		if miner.State == msgbus.OnlineState {
			if miner.Dest == cs.NodeOperator.DefaultDest {
				cs.ReadyMiners.Set(string(miners[i]), *miner)
			} else {
				cs.BusyMiners.Set(string(miners[i]), *miner)
			}
		}
	}

	// Monitor New OnlineMiners
	minerEventChan := msgbus.NewEventChan()
	_, err = cs.Ps.Sub(msgbus.MinerMsg, "", minerEventChan)
	if err != nil {
		contextlib.Logf(cs.Ctx, log.LevelError, "Failed to subscribe to miner events, Fileline::%s, Error::%v", lumerinlib.FileLine(), err)
		return err
	}
	minerMux := &sync.Mutex{}
	go cs.goMinerHandler(minerEventChan, minerMux)

	return err
}

//------------------------------------------------------------------------
//
// Monitors new contract publish events, and then subscribes to the contracts
// Then monitors the update events on the contracts, and handles state changes
//
//------------------------------------------------------------------------
func (cs *ConnectionScheduler) goContractHandler(ch msgbus.EventChan) {
	for {
		select {
		case <-cs.Ctx.Done():
			contextlib.Logf(cs.Ctx, log.LevelInfo, "Cancelling current connection scheduler context: cancelling ContractHandler go routine")
			return

		case event := <-ch:
			id := msgbus.ContractID(event.ID)

			switch event.EventType {

			//
			// Publish Event
			//
			case msgbus.PublishEvent:
				contextlib.Logf(cs.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Got Contract Publish Event: %v", event)
				contract := event.Data.(msgbus.Contract)

				if !cs.Contracts.Exists(string(id)) {
					cs.Contracts.Set(string(id), contract)
				} else {
					contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.Funcname()+"Got Publish Event, but already had the ID: %v", event)
				}

				// pusblished contract is already running
				if contract.State == msgbus.ContRunningState {
					cs.MinerUpdatedChans.Set(string(id), make(chan bool, 5))
					cs.ContractUpdatedChans.Set(string(id), make(chan bool, 5))
					if cs.Passthrough {
						go cs.ContractRunningPassthrough(id)
					} else {
						go cs.ContractRunning(id)
					}
				}

				//
				// Unpublish Event
				//
			case msgbus.UnpublishEvent:
				contextlib.Logf(cs.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Got Contract Unpublish Event: %v", event)

				if cs.Contracts.Exists(string(id)) {
					cs.Contracts.Delete(string(id))

					cs.ContractUpdatedChans.Get(string(id)).(chan bool) <- true
					close(cs.MinerUpdatedChans.Get(string(id)).(chan bool))
					close(cs.ContractUpdatedChans.Get(string(id)).(chan bool))

					cs.MinerUpdatedChans.Delete(string(id))
					cs.ContractUpdatedChans.Delete(string(id))
				} else {
					contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.Funcname()+"Got Unsubscribe Event, but dont have the ID: %v", event)
				}

				//
				// Update Event
				//
			case msgbus.UpdateEvent:
				contextlib.Logf(cs.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Got Contract Update Event: %v", event)

				if !cs.Contracts.Exists(string(id)) {
					contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.Funcname()+"Got contract ID does not exist: %v", event)
				}

				// Update the current contract data
				currentContract := cs.Contracts.Get(string(id)).(msgbus.Contract)
				cs.Contracts.Set(string(id), event.Data.(msgbus.Contract))

				if currentContract.State == event.Data.(msgbus.Contract).State {
					contextlib.Logf(cs.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Got Contract change with no state change: %v", event)
					// cs.ContractUpdatedChans.Get(string(id)).(chan bool) <- true
				} else {
					switch event.Data.(msgbus.Contract).State {
					case msgbus.ContAvailableState:
						contextlib.Logf(cs.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Found Available Contract: %v", event)
						if currentContract.State != msgbus.ContAvailableState {
							cs.ContractUpdatedChans.Get(string(id)).(chan bool) <- true
							close(cs.MinerUpdatedChans.Get(string(id)).(chan bool))
							close(cs.ContractUpdatedChans.Get(string(id)).(chan bool))

							cs.MinerUpdatedChans.Delete(string(id))
							cs.ContractUpdatedChans.Delete(string(id))
						}

					case msgbus.ContRunningState:
						contextlib.Logf(cs.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Found Running Contract: %v", event)
						if currentContract.State != msgbus.ContRunningState {
							cs.MinerUpdatedChans.Set(string(id), make(chan bool, 5))
							cs.ContractUpdatedChans.Set(string(id), make(chan bool, 5))
							if cs.Passthrough {
								go cs.ContractRunningPassthrough(id)
							} else {
								go cs.ContractRunning(id)
							}
						}

					default:
						contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.Funcname()+"Got bad state: %v", event)
					}
				}
			default:
				contextlib.Logf(cs.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Got Contract Event: %v", event)
			}
		}
	}
}

//------------------------------------------------------------------------
//
// Monitors new miner publish events, and then subscribes to the miners
// Then monitors the update events on the miners, and keeps track of their
// hashrate
//
//------------------------------------------------------------------------
func (cs *ConnectionScheduler) goMinerHandler(ch msgbus.EventChan, mux *sync.Mutex) {
	for {
		select {
		case <-cs.Ctx.Done():
			contextlib.Logf(cs.Ctx, log.LevelInfo, "Cancelling current connection scheduler context: cancelling MinerHandler go routine")
			return

		case event := <-ch:
			id := msgbus.MinerID(event.ID)

		loop:
			switch event.EventType {
			//
			// Publish Event
			//
			case msgbus.PublishEvent:
				var miner msgbus.Miner

				switch event.Data.(type) {
				case msgbus.Miner:
					miner = event.Data.(msgbus.Miner)
				case *msgbus.Miner:
					m := event.Data.(*msgbus.Miner)
					miner = *m
				}

				if miner.State != msgbus.OnlineState {
					break loop
				}

				contextlib.Logf(cs.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Got Miner Publish Event: %v", event)

				switch miner.Contract {
				case "": // no contract
					if !cs.ReadyMiners.Exists(string(id)) {
						cs.ReadyMiners.Set(string(id), miner)
						contracts := cs.Contracts.GetAll()
						for _, v := range contracts {
							if v.(msgbus.Contract).State == msgbus.ContRunningState {
								cs.MinerUpdatedChans.Get(string(v.(msgbus.Contract).ID)).(chan bool) <- true
							}
						}
					} else {
						contextlib.Logf(cs.Ctx, log.LevelPanic, "Got PubEvent, but already had the ID: %v", event)
					}
				default:
					if !cs.BusyMiners.Exists(string(id)) {
						cs.BusyMiners.Set(string(id), miner)
						contracts := cs.Contracts.GetAll()
						for _, v := range contracts {
							if v.(msgbus.Contract).State == msgbus.ContRunningState {
								cs.MinerUpdatedChans.Get(string(v.(msgbus.Contract).ID)).(chan bool) <- true
							}
						}
					} else {
						contextlib.Logf(cs.Ctx, log.LevelPanic, "Got PubEvent, but already had the ID: %v", event)
					}
				}
				contextlib.Logf(cs.Ctx, log.LevelInfo, "Ready Miners: %v", cs.ReadyMiners.M)
				contextlib.Logf(cs.Ctx, log.LevelInfo, "Busy Miners: %v", cs.BusyMiners.M)

				//
				// Update Event
				//
			case msgbus.UpdateEvent:
				var miner msgbus.Miner

				switch event.Data.(type) {
				case msgbus.Miner:
					miner = event.Data.(msgbus.Miner)
				case *msgbus.Miner:
					m := event.Data.(*msgbus.Miner)
					miner = *m
				}

				if miner.State != msgbus.OnlineState {
					cs.BusyMiners.Delete(string(id))
					cs.ReadyMiners.Delete(string(id))
					break loop
				}

				contextlib.Logf(cs.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Got Miner Update Event: %v", event)

				switch miner.Contract {
				case "": // no contract
					// Update the current miner data
					cs.BusyMiners.Delete(string(id))
					cs.ReadyMiners.Set(string(id), miner)
					contracts := cs.Contracts.GetAll()
					for _, v := range contracts {
						if v.(msgbus.Contract).State == msgbus.ContRunningState && !miner.CsMinerHandlerIgnore {
							cs.MinerUpdatedChans.Get(string(v.(msgbus.Contract).ID)).(chan bool) <- true
						}
					}
				default:
					// Update the current miner data
					cs.ReadyMiners.Delete(string(id))
					cs.BusyMiners.Set(string(id), miner)
					contracts := cs.Contracts.GetAll()
					for _, v := range contracts {
						if v.(msgbus.Contract).State == msgbus.ContRunningState && !miner.CsMinerHandlerIgnore {
							cs.MinerUpdatedChans.Get(string(v.(msgbus.Contract).ID)).(chan bool) <- true
						}
					}
				}
				contextlib.Logf(cs.Ctx, log.LevelInfo, "Ready Miners: %v", cs.ReadyMiners.M)
				contextlib.Logf(cs.Ctx, log.LevelInfo, "Busy Miners: %v", cs.BusyMiners.M)

				//
				// Unpublish Event
				//
			case msgbus.UnpublishEvent:
				contextlib.Logf(cs.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Got Miner Unpublish/Unsubscribe Event: %v", event)
				cs.BusyMiners.Delete(string(id))
				cs.ReadyMiners.Delete(string(id))
				contracts := cs.Contracts.GetAll()
				for _, v := range contracts {
					if v.(msgbus.Contract).State == msgbus.ContRunningState {
						cs.MinerUpdatedChans.Get(string(v.(msgbus.Contract).ID)).(chan bool) <- true
					}
				}

			default:
				contextlib.Logf(cs.Ctx, log.LevelTrace, lumerinlib.Funcname()+"Got Miner Event: %v", event)
			}
		}
	}
}

//------------------------------------------------------------------------
//
// Direct all miners to running contract i.e. miner hashrate calculations to service multiple contracts
//
//------------------------------------------------------------------------
func (cs *ConnectionScheduler) ContractRunningPassthrough(contractId msgbus.ContractID) {
	contextlib.Logf(cs.Ctx, log.LevelInfo, lumerinlib.FileLine()+"Contract Running in Passthrough Mode: %s", contractId)

	event, err := cs.Ps.GetWait(msgbus.ContractMsg, msgbus.IDString(contractId))
	if err != nil {
		contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", event)
	}
	contract := event.Data.(msgbus.Contract)
	destid := contract.Dest

	if destid == "" {
		contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"DestID is empty")
	}

	// Find all of the online miners point them to new target
	miners, err := cs.Ps.MinerGetAllWait()
	if err != nil {
		contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
	}

	for _, v := range miners {
		err := cs.Ps.MinerSetContractWait(v, contract.ID, destid, true)
		if err != nil {
			contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
		}
	}

	minerMapUpdated := cs.MinerUpdatedChans.Get(string(contractId)).(chan bool)
	contractUpdated := cs.ContractUpdatedChans.Get(string(contractId)).(chan bool)
	for {
		select {
		case <-cs.Ctx.Done():
			contextlib.Logf(cs.Ctx, log.LevelInfo, "Cancelling current connection scheduler context: cancelling contract running Passthrough go routine for contract: %v", contract.ID)
			return

		case <-contractUpdated:
			contextlib.Logf(cs.Ctx, log.LevelInfo, "Contract state switched to available: cancelling contract running go routine for contract: %v", contract.ID)

			// free up busy miners with this contract id
			miners := cs.BusyMiners.GetAll()
			for _, v := range miners {
				if v.(msgbus.Miner).Contract == contract.ID {
					err := cs.Ps.MinerRemoveContractWait(v.(msgbus.Miner).ID, cs.NodeOperator.DefaultDest, true)
					if err != nil {
						contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
					}
				}
			}
			return

		case <-minerMapUpdated:
			miners, err := cs.Ps.MinerGetAllWait()
			if err != nil {
				contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
			}
			for _, v := range miners {
				miner, err := cs.Ps.MinerGetWait(v)
				if err != nil {
					contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
				}
				if miner.Contract == "" {
					err = cs.Ps.MinerSetContractWait(miner.ID, contract.ID, destid, true)
					if err != nil {
						contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
					}
				}
			}
		}
	}
}

//------------------------------------------------------------------------
//
// Continously search for optimal miner combination of online miners to point to running contract
//
//------------------------------------------------------------------------
func (cs *ConnectionScheduler) ContractRunning(contractId msgbus.ContractID) {
	contextlib.Logf(cs.Ctx, log.LevelInfo, lumerinlib.FileLine()+"Contract Running: %s", contractId)

	event, err := cs.Ps.GetWait(msgbus.ContractMsg, msgbus.IDString(contractId))
	if err != nil {
		contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
	}
	contract := event.Data.(msgbus.Contract)
	hashrateTolerance := float64(contract.Limit) / 100

	availableHashrate, _ := cs.calculateHashrateAvailability(contractId)

	MIN := int(float64(contract.Speed) - float64(contract.Speed)*hashrateTolerance)

	if availableHashrate >= MIN {
		cs.SetMinerTarget(contract)
	} else {
		contextlib.Logf(cs.Ctx, log.LevelWarn, "Not enough available hashrate to fulfill contract: %v", contract.ID)

		// free up busy miners with this contract id
		miners := cs.BusyMiners.GetAll()
		for _, v := range miners {
			if v.(msgbus.Miner).Contract == contract.ID {
				err := cs.Ps.MinerRemoveContractWait(v.(msgbus.Miner).ID, cs.NodeOperator.DefaultDest, true)
				if err != nil {
					contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
				}
			}
		}
	}

	minerMapUpdated := cs.MinerUpdatedChans.Get(string(contractId)).(chan bool)
	contractUpdated := cs.ContractUpdatedChans.Get(string(contractId)).(chan bool)
	for {
		select {
		case <-cs.Ctx.Done():
			contextlib.Logf(cs.Ctx, log.LevelInfo, "Cancelling current connection scheduler context: cancelling contract running go routine for contract: %v", contract.ID)
			return

		case <-contractUpdated:
			contextlib.Logf(cs.Ctx, log.LevelInfo, "Contract state switched to available: cancelling contract running go routine for contract: %v", contract.ID)

			// free up busy miners with this contract id
			miners := cs.BusyMiners.GetAll()
			for _, v := range miners {
				if v.(msgbus.Miner).Contract == contract.ID {
					err := cs.Ps.MinerRemoveContractWait(v.(msgbus.Miner).ID, cs.NodeOperator.DefaultDest, true)
					if err != nil {
						contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
					}
				}
			}
			return

		case <-minerMapUpdated:
			//availableHashrate, contractHashrate := cs.calculateHashrateAvailability(contractId)
			availableHashrate, _ := cs.calculateHashrateAvailability(contractId)

			if availableHashrate >= MIN {
				//if contractHashrate < MIN {
				cs.SetMinerTarget(contract)
				//}
			} else {
				contextlib.Logf(cs.Ctx, log.LevelWarn, "Not enough available hashrate to fulfill contract: %v", contract.ID)
				// free up busy miners with this contract id
				miners := cs.BusyMiners.GetAll()
				for _, v := range miners {
					if v.(msgbus.Miner).Contract == contract.ID {
						err := cs.Ps.MinerRemoveContractWait(v.(msgbus.Miner).ID, cs.NodeOperator.DefaultDest, true)
						if err != nil {
							contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
						}
					}
				}
			}
		}
	}
}

func (cs *ConnectionScheduler) SetMinerTarget(contract msgbus.Contract) {
	destid := contract.Dest
	promisedHashrate := contract.Speed
	hashrateTolerance := float64(contract.Limit) / 100

	// in buyer node point miner directly to the pool
	if cs.NodeOperator.IsBuyer {
		destid = cs.NodeOperator.DefaultDest
	}

	if destid == "" {
		contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+" Error DestID is empty")
	}

	// sort miners by hashrate from least to greatest
	sortedReadyMiners := cs.sortMinersByHashrate(contract.ID)
	contextlib.Logf(cs.Ctx, log.LevelInfo, "Sorted Miners: %v", sortedReadyMiners)

	// find all miner combinations that add up to promised hashrate
	minerCombinations := findSubsets(sortedReadyMiners, promisedHashrate, hashrateTolerance)
	if minerCombinations == nil {
		contextlib.Logf(cs.Ctx, log.LevelWarn, "No valid miner combinations")

		// free up busy miners with this contract id
		miners := cs.BusyMiners.GetAll()
		for _, v := range miners {
			if v.(msgbus.Miner).Contract == contract.ID {
				err := cs.Ps.MinerRemoveContractWait(v.(msgbus.Miner).ID, cs.NodeOperator.DefaultDest, true)
				if err != nil {
					contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
				}
			}
		}
		return
	}

	contextlib.Logf(cs.Ctx, log.LevelInfo, "Valid Miner Combinations: %v", minerCombinations)

	// find best combination of miners
	minerCombination := bestCombination(minerCombinations, promisedHashrate)
	contextlib.Logf(cs.Ctx, log.LevelInfo, "Best Miner Combination: %v", minerCombination)

	// set contract and target destination for miners in optimal miner combination
	for _, v := range minerCombination {
		err := cs.Ps.MinerSetContractWait(v.id, contract.ID, destid, true)
		if err != nil {
			contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
		}
	}

	// update busy miners map with new dests based on new miner combination i.e this function was called after an update to the miner map
	newBusyMinerMap := make(map[msgbus.MinerID]Miner)
	for _, v := range minerCombination {
		newBusyMinerMap[v.id] = v
	}
	miners := cs.BusyMiners.GetAll()
	for _, v := range miners {
		if _, ok := newBusyMinerMap[v.(msgbus.Miner).ID]; !ok {
			if v.(msgbus.Miner).Contract == contract.ID {
				err := cs.Ps.MinerRemoveContractWait(v.(msgbus.Miner).ID, cs.NodeOperator.DefaultDest, true)
				if err != nil {
					contextlib.Logf(cs.Ctx, log.LevelPanic, lumerinlib.FileLine()+"Error:%v", err)
				}
			}
		}
	}
}

func (cs *ConnectionScheduler) calculateHashrateAvailability(id msgbus.ContractID) (availableHashrate int, contractHashrate int) {
	miners := cs.ReadyMiners.GetAll()
	for _, v := range miners {
		availableHashrate += v.(msgbus.Miner).CurrentHashRate
	}
	miners = cs.BusyMiners.GetAll()
	for _, v := range miners {
		if v.(msgbus.Miner).Contract == id {
			contractHashrate += v.(msgbus.Miner).CurrentHashRate
		}
	}
	availableHashrate += contractHashrate

	contextlib.Logf(cs.Ctx, log.LevelInfo, "Available Hashrate: %v", availableHashrate)

	return availableHashrate, contractHashrate
}

func (cs *ConnectionScheduler) sortMinersByHashrate(contractId msgbus.ContractID) (m MinerList) {
	m = make(MinerList, 0)

	miners := cs.ReadyMiners.GetAll()
	for _, v := range miners {
		m = append(m, Miner{v.(msgbus.Miner).ID, v.(msgbus.Miner).CurrentHashRate})
	}

	// include busy miners that are already associated with contract
	miners = cs.BusyMiners.GetAll()
	for _, v := range miners {
		if v.(msgbus.Miner).Contract == contractId {
			m = append(m, Miner{v.(msgbus.Miner).ID, v.(msgbus.Miner).CurrentHashRate})
		}
	}

	sort.Sort(m)
	return m
}

func sumSubsets(sortedMiners MinerList, n int, targetHashrate int, hashrateTolerance float64) (m MinerList) {
	// Create new array with size equal to sorted miners array to create binary array as per n(decimal number)
	x := make([]int, sortedMiners.Len())
	j := sortedMiners.Len() - 1

	// Convert the array into binary array
	for n > 0 {
		x[j] = n % 2
		n = n / 2
		j--
	}

	sum := 0

	// Calculate the sum of this subset
	for i := range sortedMiners {
		if x[i] == 1 {
			sum += sortedMiners[i].hashrate
		}
	}

	MIN := int(float64(targetHashrate) * (1 - hashrateTolerance))
	MAX := int(float64(targetHashrate) * (1 + hashrateTolerance))

	// if sum is within target hashrate bounds, subset was found
	if sum >= MIN && sum <= MAX {
		for i := range sortedMiners {
			if x[i] == 1 {
				m = append(m, sortedMiners[i])
			}
		}
		return m
	}

	return nil
}

// find subsets of list of miners whose hashrate sum equal the target hashrate
func findSubsets(sortedMiners MinerList, targetHashrate int, hashrateTolerance float64) (minerCombinations []MinerList) {
	// Calculate total number of subsets
	tot := math.Pow(2, float64(sortedMiners.Len()))

	for i := 0; i < int(tot); i++ {
		m := sumSubsets(sortedMiners, i, targetHashrate, hashrateTolerance)
		if m != nil {
			minerCombinations = append(minerCombinations, m)
		}
	}
	return minerCombinations
}

func bestCombination(minerCombinations []MinerList, targetHashrate int) MinerList {
	hashrates := make([]int, len(minerCombinations))
	numMiners := make([]int, len(minerCombinations))

	// find hashrate and number of miners in each combination
	for i := range minerCombinations {
		miners := minerCombinations[i]
		totalHashRate := 0
		num := 0
		for j := range miners {
			totalHashRate += miners[j].hashrate
			num++
		}
		hashrates[i] = totalHashRate
		numMiners[i] = num
	}

	// find combination closest to target hashrate
	index := 0
	for i := range hashrates {
		res1 := math.Abs(float64(targetHashrate) - float64(hashrates[index]))
		res2 := math.Abs(float64(targetHashrate) - float64(hashrates[i]))
		if res1 > res2 {
			index = i
		}
	}

	// if duplicate exists choose the one with the least number of miners
	newIndex := index
	for i := range hashrates {
		if hashrates[i] == hashrates[index] && numMiners[i] < numMiners[newIndex] {
			newIndex = i
		}
	}

	return minerCombinations[newIndex]
}
