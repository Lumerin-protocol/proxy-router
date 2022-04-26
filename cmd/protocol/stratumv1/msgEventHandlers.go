package stratumv1

import (
	"fmt"

	simple "gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/SIMPL"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

//
//
//
func (svs *StratumV1Struct) handleMsgBusEvent(event *simple.SimpleMsgBusEvent) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called, Request ID:%d", event.RequestID)

	switch event.EventType {
	case simple.NoEvent:
		fmt.Printf(lumerinlib.Funcname() + " NoEvent received, returning\n")
		return
	case simple.MsgUpdateEvent:
		svs.handleMsgUpdateEvent(event)
		return
	case simple.MsgDeleteEvent:
		svs.handleMsgDeleteEvent(event)
		return
	case simple.MsgGetEvent:
		svs.handleMsgGetEvent(event)
		return
	case simple.MsgGetIndexEvent:
		svs.handleMsgIndexEvent(event)
		return
	case simple.MsgSearchEvent:
		svs.handleMsgSearchEvent(event)
		return
	case simple.MsgSearchIndexEvent:
		svs.handleMsgSearchIndexEvent(event)
		return
	case simple.MsgPublishEvent:
		svs.handleMsgPublishEvent(event)
		return
	case simple.MsgUnpublishEvent:
		svs.handleMsgUnpublishEvent(event)
		return
	case simple.MsgSubscribedEvent:
		svs.handleMsgSubscribedEvent(event)
		return
	case simple.MsgUnsubscribedEvent:
		svs.handleMsgUnsubscribedEvent(event)
		return
	case simple.MsgRemovedEvent:
		svs.handleMsgRemovedEvent(event)
		return

	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Unknown event message:%s:%s", event.EventType, event.ID)
	}
}

//
// handleMsgUpdateEvent()
// Check the request ID to see if it was requested, pull the request off and handle the result
// else
// Check to see if this is a event that we care about, and handle it
//
func (svs *StratumV1Struct) handleMsgUpdateEvent(event *simple.SimpleMsgBusEvent) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	// Parse Event Msg

	var minerrec msgbus.Miner
	switch event.Msg {
	case simple.MinerMsg:
		switch event.Data.(type) {
		case *msgbus.Miner:
			minerrecptr := event.Data.(*msgbus.Miner)
			minerrec = *minerrecptr
		case msgbus.Miner:
			minerrec = event.Data.(msgbus.Miner)
		default:
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" event.Data is not a msgbus.Miner struct")
		}

		// Check that we recieved the correct miner ID from the msgbus
		if svs.minerRec.ID != minerrec.ID {
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" event.Data records do not match up")
		}

		// Did the Dest ID change?
		if svs.minerRec.Dest != minerrec.Dest {
			contextlib.Logf(svs.Ctx(), contextlib.LevelInfo, lumerinlib.FileLineFunc()+" Miner:%s Dest changed to: %s from %s", minerrec.ID, minerrec.Dest, svs.minerRec.Dest)

			// Destination has changed...
			switch_to_uid := svs.GetDstUIDDestID(minerrec.Dest)

			svs.switchToDestID = minerrec.Dest

			// Is Dest already open and running?
			// No, open it
			if switch_to_uid < 0 {
				// Start the process of transitioning to a new Dest
				_, e := svs.protocol.Get(simple.DestMsg, simple.IDString(minerrec.Dest))
				if e != nil {
					contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Get() error:%s", e)
				}
			}

		} else {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Recieved Miner update, but dest did not change:%s:%s", event.EventType, event.ID)
		}

	// Ignore all others
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Unknown event message:%s:%s", event.EventType, event.ID)
	}

}

//
// handleMsgDeleteEvent()
// Check the request ID to see if it was requested, pull the request off and handle the result
// else
// Check to see if this is a event that we care about, and handle it
//
func (svs *StratumV1Struct) handleMsgDeleteEvent(event *simple.SimpleMsgBusEvent) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	switch event.Msg {
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Unknown event message:%s:%s", event.EventType, event.ID)
	}

}

//
// handleMsgGetEvent()
// Check the request ID to see if it was requested, pull the request off and handle the result
// else error out
//
func (svs *StratumV1Struct) handleMsgGetEvent(event *simple.SimpleMsgBusEvent) {
	if svs == nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" svs is nil")
	}

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	if event.Err != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Event Err:%s", event.Err)
		return
	}

	if event.Data == nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Event Data is nil")
		return
	}

	switch event.Msg {
	case simple.DestMsg:
		// Recieved a Dest Get()
		// It would be used to setup the next dest connection
		if event.ID == simple.IDString(svs.switchToDestID) {
			// Fire up the default destination connction here
			// If default is not already set, set it
			var dest msgbus.Dest
			switch event.Data.(type) {
			case msgbus.Dest:
				dest = event.Data.(msgbus.Dest)
			case *msgbus.Dest:
				d := event.Data.(*msgbus.Dest)
				if d == nil {
					contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" DestMsg: is nil")
				}
				dest = *d
			default:
				contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" DestMsg: bad data:%t", event.Data)
			}

			id := dest.ID

			uid := svs.GetDstUIDDestID(id)
			if 0 > uid {
				// Open new Dest
				svs.SetDstStateUid(uid, DstStateDialing)
				e := svs.protocol.AsyncDial(&dest)
				if e != nil {
					contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" AsyncDial returned error:%s", e)
				}
			} else {
				contextlib.Logf(svs.Ctx(), contextlib.LevelWarn, lumerinlib.FileLineFunc()+" Dest already opened:%s", event.ID)
			}
		} else {
			contextlib.Logf(svs.Ctx(), contextlib.LevelWarn, lumerinlib.FileLineFunc()+" Recieved Dest Get(), but it does not match next dest, dropping it:%s", event.ID)
		}

	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Unknown event message:%s:%s", event.EventType, event.ID)
	}

}

//
// handleMsgIndexEvent()
// Check the request ID to see if it was requested, pull the request off and handle the result
// else error out
//
func (svs *StratumV1Struct) handleMsgIndexEvent(event *simple.SimpleMsgBusEvent) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	switch event.Msg {
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Unknown event message:%s:%s", event.EventType, event.ID)
	}

}

//
// handleMsgSearchEvent()
// Check the request ID to see if it was requested, pull the request off and handle the result
// else error out
//
func (svs *StratumV1Struct) handleMsgSearchEvent(event *simple.SimpleMsgBusEvent) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	switch event.Msg {
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Unknown event message:%s:%s", event.EventType, event.ID)
	}

}

//
// handleMsgSearchEvent()
// Check the request ID to see if it was requested, pull the request off and handle the result
// else error out
//
func (svs *StratumV1Struct) handleMsgSearchIndexEvent(event *simple.SimpleMsgBusEvent) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	switch event.Msg {
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Unknown event message:%s:%s", event.EventType, event.ID)
	}

}

//
// handleMsgPublishEvent()
// Check the request ID to see if it was requested, pull the request off and handle the result
// else
// Check to see if this is a event that we care about, and handle it
//
func (svs *StratumV1Struct) handleMsgPublishEvent(event *simple.SimpleMsgBusEvent) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	switch event.Msg {
	case simple.MinerMsg:
		contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" event message:%s:%s, subscribe", event.EventType, event.ID)

		_, e := svs.protocol.Sub(simple.MinerMsg, event.ID)
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" event message:%s:%s, subscribe", event.EventType, event.ID)
		}

	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Unknown event message:%s:%s", event.EventType, event.ID)
	}

}

//
// handleMsgUnpublishEvent()
// Check the request ID to see if it was requested, pull the request off and handle the result
// else
// Check to see if this is a event that we care about, and handle it
//
func (svs *StratumV1Struct) handleMsgUnpublishEvent(event *simple.SimpleMsgBusEvent) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	switch event.Msg {
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Unknown event message:%s:%s", event.EventType, event.ID)
	}

}

//
// handleMsgSubscribedEvent()
// Check the request ID to see if it was requested, pull the request off and handle the result
// else
// Check to see if this is a event that we care about, and handle it
//
func (svs *StratumV1Struct) handleMsgSubscribedEvent(event *simple.SimpleMsgBusEvent) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	switch event.Msg {
	case simple.MinerMsg:
		contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Subscribed event message:%s:%s", event.EventType, event.ID)

	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Unknown event message:%s:%s", event.EventType, event.ID)
	}

}

//
// handleMsgUnisubscribedEvent()
// Check the request ID to see if it was requested, pull the request off and handle the result
// else
// Check to see if this is a event that we care about, and handle it
//
func (svs *StratumV1Struct) handleMsgUnsubscribedEvent(event *simple.SimpleMsgBusEvent) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	switch event.Msg {
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Unknown event message:%s:%s", event.EventType, event.ID)
	}

}

//
// handleMsgRemovedEvent()
// Check the request ID to see if it was requested, pull the request off and handle the result
// else
// Check to see if this is a event that we care about, and handle it
//
//
func (svs *StratumV1Struct) handleMsgRemovedEvent(event *simple.SimpleMsgBusEvent) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	switch event.Msg {
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Unknown event message:%s:%s", event.EventType, event.ID)
	}

}
