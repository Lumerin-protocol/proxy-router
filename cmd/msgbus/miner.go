package msgbus

import (
	"fmt"

	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

const (
	OnlineState  MinerState = "MinerOnlineState"
	OfflineState MinerState = "MinerOfflineState"
)

type MinerID IDString

//
// Created & Updated by Connection Manager
//
type Miner struct {
	ID                      MinerID
	Name                    string
	IP                      string
	Port                    int
	MAC                     string
	State                   MinerState
	Contracts               map[ContractID]float64 // Updated by Connection Scheduler
	Dest                    DestID     // Updated by Connection Scheduler
	InitialMeasuredHashRate int
	CurrentHashRate         int 
	TimeSlice				bool
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) MinerPubWait(miner Miner) (m Miner, err error) {
	if miner.ID == "" {
		miner.ID = MinerID(GetRandomIDString())
	}

	event, err := ps.PubWait(MinerMsg, IDString(miner.ID), miner)
	if err != nil || event.Err != nil {
		panic(fmt.Sprintf(lumerinlib.Funcname()+" Unable to add Record %s, %s\n", err, event.Err))
	}

	m = event.Data.(Miner)
	if err != nil || event.Err != nil {
		fmt.Printf(lumerinlib.Funcname()+" PubWait returned err: %s, %s\n", err, event.Err)
	}

	return m, err
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) MinerGetWait(id MinerID) (miner *Miner, err error) {
	event, err := ps.GetWait(MinerMsg, IDString(id))
	if err != nil || event.Err != nil {
		fmt.Printf(lumerinlib.Funcname()+" ID not found %s, %s\n", err, event.Err)
	}

	switch event.Data.(type) {
	case Miner:
		m := event.Data.(Miner)
		miner = &m
	case *Miner:
		miner = event.Data.(*Miner)
	default:
		miner = nil
	}

	return miner, err
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) MinerSetWait(miner Miner) (err error) {
	if miner.ID == "" {
		panic(fmt.Sprintf(lumerinlib.Funcname() + " ID not provided\n"))
	}

	_, err = ps.MinerGetWait(miner.ID)
	if err != nil {
		return err
	}

	e, err := ps.SetWait(MinerMsg, IDString(miner.ID), miner)
	if err != nil {
		return err
	}

	if e.Err != nil {
		return e.Err
	}

	return nil

}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) MinerGetAllWait() (miners []MinerID, err error) {
	event, err := ps.GetWait(MinerMsg, "")
	if err != nil || event.Err != nil {
		fmt.Printf(lumerinlib.Funcname()+" Error gettig all  %s, %s\n", err, event.Err)
		if err != nil {
			return nil, err
		} else {
			return nil, event.Err
		}
	}

	if event.EventType != GetIndexEvent {
		panic(fmt.Sprint(lumerinlib.FileLine()+" Error:%v\n", event))
	}

	count := len(event.Data.(IDIndex))
	miners = make([]MinerID, count)

	if count == 0 {
		fmt.Printf(lumerinlib.FileLine() + " No miners are online\n")
	} else {
		for i, v := range event.Data.(IDIndex) {
			miners[i] = MinerID(v)
		}
	}
	return miners, err
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) MinerExistsWait(id MinerID) bool {
	miner, _ := ps.MinerGetWait(id)
	return miner != nil
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) MinerSetDestWait(miner MinerID, dest DestID) (m *Miner, err error) {
	m, err = ps.MinerGetWait(miner)
	if err != nil {
		fmt.Printf(lumerinlib.FileLine()+" MinerGetWait errored out:%s\n", err)
		return m,err
	} else {
		m.Dest = dest
		err = ps.MinerSetWait(*m)
		if err != nil {
			fmt.Printf(lumerinlib.FileLine()+" MinerSetWait errored out:%s\n", err)
		}
	}
	return m,err
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) MinerSetContractWait(miner MinerID, contract ContractID, slicePercent float64, timeSlice bool) (m *Miner, err error) {
	m, err = ps.MinerGetWait(miner)
	if err != nil {
		fmt.Printf(lumerinlib.FileLine()+" MinerGetWait errored out:%s\n", err)
		return m, err
	} else {
		m.Contracts[contract] = slicePercent
		m.TimeSlice = timeSlice
		err = ps.MinerSetWait(*m)
		if err != nil {
			fmt.Printf(lumerinlib.FileLine()+" MinerSetWait errored out:%s\n", err)
		}
	}
	return m, err
}

func (ps *PubSub) MinerRemoveContractWait(miner MinerID, contract ContractID, defaultDest DestID) (m *Miner, err error) {
	m, err = ps.MinerGetWait(miner)
	if err != nil {
		fmt.Printf(lumerinlib.FileLine()+" MinerGetWait errored out:%s\n", err)
		return m, err
	} else {
		if _,ok := m.Contracts[contract]; !ok {
			fmt.Println(lumerinlib.FileLine()+"Trying to remove contract from Miner that doesn't contain it")
		}
		delete(m.Contracts, contract)
		if len(m.Contracts) == 0 {
			m.Dest = defaultDest
			m.TimeSlice = false
		} else {
			sliced := false
			loop:
			for _,c := range m.Contracts {
				if c < 1 {
					sliced = true
					break loop
				}
			}
			m.TimeSlice = sliced
		}
		err = ps.MinerSetWait(*m)
		if err != nil {
			fmt.Printf(lumerinlib.FileLine()+" MinerSetWait errored out:%s\n", err)
		}
	}
	return m, err
}

func (ps *PubSub) MinersContainContract(contract ContractID) (result []Miner) {
	miners,err := ps.MinerGetAllWait()
	if err != nil {
		panic(fmt.Sprintf(lumerinlib.Funcname()+" Error gettig all miners, error %v\n", err))
	}
	for _,m := range miners {
		miner,err := ps.MinerGetWait(m)
		if err != nil {
			panic(fmt.Sprintf(lumerinlib.Funcname()+" Error gettig miner, error %v\n", err))
		}
		if _,ok := miner.Contracts[contract]; ok {	
			result = append(result, *miner)
		}
	}
	return result
}

func (ps *PubSub) MinerSlicedUtilization(id MinerID) float64 {
	miner,err := ps.MinerGetWait(id)
	if err != nil {
		panic(fmt.Sprintf(lumerinlib.Funcname()+" Error gettig miner, error %v\n", err))
	}
	var contractSlicedPercent float64
	for _,v := range miner.Contracts {
		contractSlicedPercent += v
	}

	return (1 - contractSlicedPercent)
}