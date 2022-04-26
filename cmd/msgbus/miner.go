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
	MAC                     string
	State                   MinerState
	Contract                ContractID // Updated by Connection Scheduler
	Dest                    DestID     // Updated by Connection Scheduler
	InitialMeasuredHashRate int
	CurrentHashRate         int
	CsMinerHandlerIgnore    bool // Ignore update in Connection Scheduler Miner Handler
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
func (ps *PubSub) MinerSetDestWait(miner MinerID, dest DestID) (err error) {

	m, err := ps.MinerGetWait(miner)
	if err != nil {
		fmt.Printf(lumerinlib.FileLine()+" MinerGetWait errored out:%s\n", err)
		return err
	} else {
		m.Dest = dest
		err = ps.MinerSetWait(*m)
		if err != nil {
			fmt.Printf(lumerinlib.FileLine()+" MinerSetWait errored out:%s\n", err)
		}
	}

	return err
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) MinerSetContractWait(miner MinerID, contract ContractID, targetDest DestID, csIgnore bool) (err error) {
	m, err := ps.MinerGetWait(miner)
	if err != nil {
		fmt.Printf(lumerinlib.FileLine()+" MinerGetWait errored out:%s\n", err)
		return err
	} else {
		m.Contract = contract
		m.Dest = targetDest
		m.CsMinerHandlerIgnore = csIgnore
		err = ps.MinerSetWait(*m)
		if err != nil {
			fmt.Printf(lumerinlib.FileLine()+" MinerSetWait errored out:%s\n", err)
		}
	}

	return err
}

func (ps *PubSub) MinerRemoveContractWait(miner MinerID, defaultDest DestID, csIgnore bool) (err error) {
	m, err := ps.MinerGetWait(miner)
	if err != nil {
		fmt.Printf(lumerinlib.FileLine()+" MinerGetWait errored out:%s\n", err)
		return err
	} else {
		m.Contract = ""
		m.Dest = defaultDest
		m.CsMinerHandlerIgnore = csIgnore
		err = ps.MinerSetWait(*m)
		if err != nil {
			fmt.Printf(lumerinlib.FileLine()+" MinerSetWait errored out:%s\n", err)
		}
	}

	return err
}
