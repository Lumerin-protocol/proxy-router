package stratumv1

import (
	"fmt"

	simple "gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/SIMPL"
	"gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/connectionmanager"
	"gitlab.com/TitanInd/lumerin/cmd/protocol"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

//
// States:
// Transition to a new Destination ID
//		Is the Dest Open, if not open it
//		Set State so when the source is ready it can transition
//
//

//
// handleConnOpenEvent()
// UID is the unique ID of the DST connection
// UID is what is fed to the connection functions like Write
// New connection starts off in the new state, immediatly send a Subscribe message, then Auth
//
func (svs *StratumV1Struct) handleConnOpenEvent(scoe *simple.SimpleConnOpenEvent) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" called")

	//
	// Need Requirements For This: Error on connection
	// If there is an error do we handle it here, or put it into the connection struct?
	//
	e = scoe.Err()
	if nil != e {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" scre had an error:%v", scoe)
		return e
	}

	uid := scoe.UniqueID()
	dest := scoe.Dest()

	// Need a new Dst Conn Connection Record.

	e = svs.protocol.GetDstStruct().NewProtocolDstStruct(scoe)
	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" NewProtocolDstStruct() error:%s", e)
		return e
	}
	svs.dstState[uid] = DstStateNew
	svs.dstDest[uid] = dest

	dstconn, e := svs.protocol.GetDstConn(uid)
	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" GetDstConn() bad index:%d", uid)
		return e
	}
	dstconn.SetState(protocol.ConnStateReady)

	// Send initialization subscribe message here
	// Set state to DstStateSubscribing

	request := svs.srcSubscribeRequest

	msg, e := request.createRequestMsg()
	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" GetDstConn() bad UID:%d", uid)
	}

	LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_STOR2DST, msg)

	msgsize := len(msg)

	if e == nil {
		count, e := svs.protocol.WriteDst(uid, msg)
		if e != nil {
			svs.SetDstStateUid(uid, DstStateError)
		}

		if count != msgsize {
			svs.SetDstStateUid(uid, DstStateError)
		}

		svs.SetDstStateUid(uid, DstStateSubscribing)
	}

	return e
}

//
// handleConnReadEvent()
// Look up the connection index
// Parse the message
// Run the message through a handle routine
//
func (svs *StratumV1Struct) handleConnReadEvent(scre *simple.SimpleConnReadEvent) (e error) {

	uid := scre.UniqueID()

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called UID:%d", uid)

	// Validate the index is good HERE

	e = scre.Err()
	if nil != e {
		switch e {
		case connectionmanager.ErrConnMgrClosed:
			contextlib.Logf(svs.Ctx(), contextlib.LevelDebug, lumerinlib.FileLineFunc()+" Connection Manager Closed, closing down the stratum connection here")
			svs.Cancel()
			return nil
		case connectionmanager.ErrConnDstClosed:
			contextlib.Logf(svs.Ctx(), contextlib.LevelDebug, lumerinlib.FileLineFunc()+" Dst:%d Closed -> Redialing", uid)
			svs.SetDstStateUid(uid, DstStateRedialing)
			e = svs.protocol.AsyncReDial(uid)
			return e
		default:
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Dst:%d  Error not handled:%s", uid, e)
		}

		// Notate it here, but set the err in the connection later
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" scre had an error:%s", e)
	}

	//
	// Should be moved into protocol.go file since this is only protocol related.
	//
	var pcs *protocol.ProtocolConnectionStruct
	if uid < 0 {
		pcs, e = svs.protocol.GetSrcConn()
	} else {
		pcs, e = svs.protocol.GetDstConn(uid)
	}

	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" GetDstConn(%d) had an error:%s", uid, e)
	}

	// Buffer data locally
	data := scre.Data()
	if len(data) == 0 {
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" scre had no data")
	}

	pcs.AddBuf(data)

	// -- Move above code to Protocol section --

	// If there is a '\n' in the message, extract the message
	// Adjust the buffer accordingly

	// Cycle through the buffer until all of the '\n' are found
	for buf, e := pcs.GetLineTermData(); len(buf) > 0 && e == nil; buf, e = pcs.GetLineTermData() {

		if len(buf) == 0 {
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" buffer is 0")
		}

		// Process the message
		ret, e := unmarshalMsg(buf)

		if uid < 0 {
			LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_RECV_SRC, buf)
		} else {
			LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_RECV_DST, buf)
		}

		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Called data:%s", data)
			return e
		}

		switch ret := ret.(type) {
		case *stratumRequest:
			e = svs.handleRequest(uid, ret)
		case *stratumResponse:
			e = svs.handleResponse(uid, ret)
		case *stratumNotice:
			e = svs.handleNotice(uid, ret)
		default:
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Called")
		}
		if e != nil {
			break
		}
	}

	return e
}

//
// handleConnEOFEvent()
// Close out the connection index
// Could reopen the connection, or perform some error handling
//
func (svs *StratumV1Struct) handleConnEOFEvent(event *simple.SimpleEvent) {

	// contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Not Handled Yet!")

}

//
// handleConnErrorEvent()
// Perform some error handling
//
func (svs *StratumV1Struct) handleConnErrorEvent(event *simple.SimpleEvent) {

	// contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Not Handled Yet!")
}

//
// handleErrorEvent()
// Perform Error handling
//
func (svs *StratumV1Struct) handleErrorEvent(event *simple.SimpleEvent) {

	// contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Not Handled Yet!")
}

//
// handleRequest()
// index: -1 = SRC, 0 = default, >0 = Dst
//
func (svs *StratumV1Struct) handleRequest(uid simple.ConnUniqueID, request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called Index: %d Method: %s", uid, request.Method)

	// Recieved Src message
	if uid < 0 {
		switch request.Method {
		case string(CLIENT_MINING_CAPABILITIES):
			svs.handleSrcReqCapabilities(request)
		case string(CLIENT_MINING_EXTRANONCE):
			svs.handleSrcReqExtranonce(request)
		case string(CLIENT_MINING_AUTHORIZE):
			e = svs.handleSrcReqAuthorize(request)
		case string(CLIENT_MINING_CONFIGURE):
			e = svs.handleSrcReqConfigure(request)
		case string(CLIENT_MINING_SUBSCRIBE):
			e = svs.handleSrcReqSubscribe(request)
		case string(CLIENT_MINING_SUBMIT):
			svs.handleSrcReqSubmit(request)
		case string(CLIENT_MINING_SUGGEST_DIFFICULTY):
			svs.handleSrcReqDifficulty(request)
		case string(CLIENT_MINING_SUGGEST_TARGET):
			svs.handleSrcReqSuggestTarget(request)
		case string(CLIENT_MINING_MULTI_VERSION):
			contextlib.Logf(svs.Ctx(), contextlib.LevelInfo, lumerinlib.FileLineFunc()+" Recieved:%s, IGNORING", request.Method)
		default:
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Bad Destination Message Type Recieved:%s", request.Method)
		}

		// Received Dst message
	} else {
		switch request.Method {
		case string(SERVER_MINING_NOTIFY):
			svs.handleDstReqNotify(uid, request)
		case string(SERVER_GET_VERSION):
			svs.handleDstReqGetVersion(uid, request)
		case string(SERVER_SHOW_MESSAGE):
			svs.handleDstReqShowMessage(uid, request)
		case string(SERVER_MINING_PING):
			svs.handleDstReqPing(uid, request)
		case string(SERVER_MINING_SET_EXTRANONCE):
			svs.handleDstReqSetExtranonce(uid, request)
		case string(SERVER_MINING_SET_GOAL):
			svs.handleDstReqSetGoal(uid, request)
		case string(SERVER_MINING_SET_DIFFICULTY):
			svs.handleDstReqSetDifficulty(uid, request)
		default:
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Bad Destination Message Type Recieved:%s", request.Method)
		}
	}

	return e
}

//
// handleResponse()
// index: -1 = SRC, 0 = default, >0 = Dst
//
func (svs *StratumV1Struct) handleResponse(uid simple.ConnUniqueID, response *stratumResponse) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called UID:%d", uid)

	//
	// SRC connection
	//
	if uid < 0 {

		srcstate := svs.GetSrcState()

		if response.Error != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Src State:%s, error:%s, %v", srcstate, response.Error, response)
			//
			// Which returned errors should result in being put into an Error state?
			// Log it for now.
			// svs.SetSrcState(SrcStateError)
			// srcstate = SrcStateError
			//
		}

		switch srcstate {
		case SrcStateSubscribed:
		case SrcStateError:
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Src is in Error State, closing connection")
			svs.Close()

		default:
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" not handled yet")
		}

		//
		// DST connection
		//
	} else {

		dststate := svs.GetDstStateUid(uid)

		contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" DST UID:%d State:%s", uid, dststate)

		// Notate the Error, and pass it on to the miner
		if response.Error != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Dst UID:%d, State:%s, error:%s, %v", uid, dststate, *response.Error, response)
			// svs.SetDstStateUid(uid, DstStateError)
			// dststate = DstStateError
		}

		switch dststate {
		//
		// Got Response to Subscribe, send authorize now
		// drop the response message
		// push the auth message back
		//
		case DstStateSubscribing:

			// Did I get a response to the subscribe here?
			if response.ID != svs.srcSubscribeRequest.ID {
				contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" createResponeMsg() response.ID:%d, Subscrb.ID:%d", response.ID, svs.srcSubscribeRequest.ID)
				return nil
			}

			//
			// Got Response to Subscribe, pick out the extranonce and extranonce2 values and store them
			//
			if response.Error == nil {
				intarray := response.Result.([]interface{})
				extranonce := intarray[1].(string)
				extranonce2size := int(intarray[2].(float64))

				svs.dstExtranonce[uid] = extranonce
				svs.dstExtranonce2size[uid] = extranonce2size

			}

			//
			//  Now sent Authorize
			//
			svs.SetDstStateUid(uid, DstStateAuthorizing)
			request := svs.srcAuthRequest
			username := svs.dstDest[uid].Username()
			password := svs.dstDest[uid].Password()

			msg, e := request.createAuthorizeRequestMsg(username, password)

			LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_STOR2DST, msg)
			if e != nil {
				contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" error:%s", e)
			}
			count, e := svs.protocol.WriteDst(uid, msg)
			if e != nil {
				contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" error:%s", e)
			}
			if count != len(msg) {
				contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" count:%d, len:%d", count, len(msg))
			}

		//
		// Got Response to Authorize, Set state to running now
		// drop the response message
		//
		case DstStateAuthorizing:

			// Did I get a response to the subscribe here?
			if response.ID != svs.srcAuthRequest.ID {
				contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" createResponeMsg() response.ID:%d, Auth.ID:%d", response.ID, svs.srcAuthRequest.ID)
				return nil
			}

			svs.SetDstStateUid(uid, DstStateStandBy)

			if svs.scheduler == OnDemand {
				svs.switchDest()
			}

		//
		// Pass response messages when in Running State
		//
		case DstStateRunning:

			msg, e := response.createResponseMsg()
			if e != nil {
				contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" createResponseMsg() error:%s", e)
			}

			// Write to the current destination

			LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_DST2SRC, msg)

			count, e := svs.protocol.WriteSrc(msg)
			if e != nil {
				contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" WriteSrc error:%s", e)
				return e
			}
			if count != len(msg) {
				contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" WriteSrc bad count:%d, %d", count, len(msg))
				e = fmt.Errorf(lumerinlib.FileLineFunc()+" WriteSrc bad count:%d, %d", count, len(msg))
				return e
			}

			//
			// Send configure Here?
			//

		case DstStateStandBy:

			msg, e := response.createResponseMsg()
			if e != nil {
				contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" createResponseMsg() error:%s", e)
			}
			LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_DROP_DST, msg)

			contextlib.Logf(svs.Ctx(), contextlib.LevelDebug, lumerinlib.FileLineFunc()+" state not handled yet:%s", dststate)

		case DstStateError:

			e = svs.DstRedialUid(uid)
			if e != nil {
				contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Dst is in Error State, closing connection")
				svs.CloseUid(uid)
			}

		case DstStateClosed:
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Connecton is already closed")

		default:
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" not handled yet")
		}

	}

	return nil
}

//
// handleNotice()
// index: -1 = SRC, 0 = default, >0 = Dst
//
func (svs *StratumV1Struct) handleNotice(uid simple.ConnUniqueID, notice *stratumNotice) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" Called")

	if uid < 0 {
		switch notice.Method {
		default:
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Bad Destination Message Type Recieved")
		}
	} else {

		switch notice.Method {
		case string(SERVER_MINING_NOTIFY):
			svs.handleDstNoticeNotify(uid, notice)
		case string(SERVER_MINING_SET_EXTRANONCE):
			svs.handleDstNoticeSetExtranonce(uid, notice)
		case string(SERVER_MINING_SET_DIFFICULTY):
			svs.handleDstNoticeSetDifficulty(uid, notice)
		//case string(SERVER_MINING_SET_VERSION_MASK):
		//	svs.handleDstNoticeSetVersionMask(uid, notice)
		case string(SERVER_RECONNECT):
			svs.handleDstNoticeReconnect(uid, notice)
		default:
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Bad Destination Message Type Recieved:%s", notice.Method)
		}
	}

	return nil
}

// -------------------------------------------------------------------------
// Incoming Conn SRC REQUESTs
// -------------------------------------------------------------------------

//
// handleSrcReqSubscribe()
// State Transition: SrcStateNew -> SrcStateSubscribed
//
func (svs *StratumV1Struct) handleSrcReqSubscribe(request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" enter")

	state := svs.GetSrcState()

	// Validate the current sstate of the SRC connection
	switch state {
	case SrcStateNew:
		// This is what we expect, so skip
	case SrcStateSubscribed:
		// Returing subscribe, something went wrong.
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Got Subscribe, but already subscribed")
		return ErrBadSrcState
	case SrcStateAuthorized:
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Got Subscribe, but already authorized")
		return ErrBadSrcState
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Src state:%s", state)
	}

	if svs.srcSubscribeRequest == nil {
		r := *request
		svs.srcSubscribeRequest = &r
		svs.srcSubscribeRequest.ID = 1
		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_STOR_SRC, svs.srcSubscribeRequest)
	}

	svs.SetSrcState(SrcStateSubscribed)

	dst := contextlib.GetDest(svs.Ctx())
	if dst == nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" GetDest() returned nil")
	}

	response := &stratumResponse{}
	msg, e := response.createSrcSubscribeResponseMsg(request.ID)
	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" createResponseMsg error:%s", e)
		return e
	}

	LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_STOR2SRC, msg)

	count, e := svs.protocol.WriteSrc(msg)
	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" WriteSrc error:%s, Close it down", e)
		// Error writing to Src (close it down here)
		svs.Cancel()
		return e
	}
	if count != len(msg) {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" WriteSrc bad count:%d, %d", count, len(msg))
		e = fmt.Errorf(lumerinlib.FileLineFunc()+" WriteSrc bad count:%d, %d", count, len(msg))
		return e
	}

	return nil
}

//
// handleSrcReqAuthorize()
// State Transition: SrcStateSubscribed -> SrcStateAuthorized
// Open Comm to Default Dst Here
//
func (svs *StratumV1Struct) handleSrcReqAuthorize(request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" enter")

	state := svs.GetSrcState()
	// Validate the current sstate of the SRC connection
	switch state {
	case SrcStateNew:
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Got Authorize, expect subscribe")
		return ErrBadSrcState
	case SrcStateSubscribed:
		// This is what we expect, so continue
	case SrcStateAuthorized:
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Got Authorize, but already authorized")
		return ErrBadSrcState
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Src state:%s", state)
	}

	if svs.srcAuthRequest == nil {
		r := *request
		svs.srcAuthRequest = &r
		svs.srcAuthRequest.ID = svs.srcSubscribeRequest.ID + 1
		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_STOR_SRC, svs.srcAuthRequest)
	}
	svs.SetSrcState(SrcStateAuthorized)

	//id, e := request.getID()
	//if e != nil {
	//	contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" getID() returned error:%s ", e)
	//	return e
	//}

	dstID := contextlib.GetDest(svs.Ctx())
	if dstID == nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" GetDest() returned nil")
	}

	// Move this to JSON file

	result := true

	response := &stratumResponse{
		ID:     request.ID,
		Error:  nil,
		Result: result,
		Reject: nil,
	}

	msg, e := response.createResponseMsg()
	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" createResponseMsg error:%s", e)
		return e
	}

	LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_STOR2SRC, msg)

	count, e := svs.protocol.WriteSrc(msg)
	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" WriteSrc error:%s", e)
		return e
	}
	if count != len(msg) {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" WriteSrc bad count:%d, %d", count, len(msg))
		e = fmt.Errorf(lumerinlib.FileLineFunc()+" WriteSrc bad count:%d, %d", count, len(msg))
		return e
	}

	//
	// Sets up mining record in the MsgBus
	//
	svs.newMinerRecordPub()

	//
	// Open up the default pool connection
	//
	svs.openDefaultConnection()

	return nil
}

//
// handleSrcReqConfigure()
//
func (svs *StratumV1Struct) handleSrcReqConfigure(request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" enter")

	state := svs.GetSrcState()
	// Validate the current state of the SRC connection
	switch state {
	case SrcStateNew:
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Got Configure")
		// return ErrBadSrcState
	case SrcStateSubscribed:
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Got Configure")
		// return ErrBadSrcState
	case SrcStateAuthorized:
	case SrcStateRunning:
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Src state:%s", state)
	}

	//
	// Get the current default route UID
	//
	uid, _ := svs.protocol.GetDefaultRouteUID()
	if uid < 0 {
		// Need to store this for later use
		msg, e := request.createRequestMsg()
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" createRequestMsg error:%s", e)
			return e
		}

		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_STOR_SRC, msg)

		svs.srcConfigure = request

		response := &stratumResponse{
			ID:     request.ID,
			Error:  nil,
			Result: nil,
			Reject: nil,
		}

		respmsg, e := response.createSrcConfigureResponseMsg()
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" createResponsMsg() error:%s", e)
			return e
		}

		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_STOR2SRC, respmsg)

		count, e := svs.protocol.WriteSrc(respmsg)
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Write error:%s", e)
			return e
		}
		if count != len(respmsg) {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Write bad count:%d, %d", count, len(msg))
			e = fmt.Errorf(lumerinlib.FileLineFunc()+" WriteSrc bad count:%d, %d", count, len(msg))
			return e
		}

		r := *request
		svs.srcConfigure = &r

		return e
	} else {

		// Move this to JSON file

		msg, e := request.createRequestMsg()
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" createRequestMsg error:%s", e)
			return e
		}

		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_SRC2DST, msg)
		count, e := svs.protocol.Write(msg)
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Write error:%s", e)
			return e
		}
		if count != len(msg) {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Write bad count:%d, %d", count, len(msg))
			e = fmt.Errorf(lumerinlib.FileLineFunc()+" WriteSrc bad count:%d, %d", count, len(msg))
			return e
		}

	}
	return nil
}

//
// handleSrcReqSubmit()
// The system should be authorized at least, possibly other states too
//
func (svs *StratumV1Struct) handleSrcReqSubmit(request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" enter")

	state := svs.GetSrcState()
	// Validate the current sstate of the SRC connection
	switch state {
	case SrcStateNew:
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Got Submit, expecting Subscribe")
		return ErrBadSrcState
	case SrcStateSubscribed:
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Got Submit, expecting Authorize")
		return ErrBadSrcState
	case SrcStateAuthorized:
		svs.SetSrcState(SrcStateRunning)
	case SrcStateRunning:
		// This is what we expect, so skip
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Src state:%s", state)
	}

	//
	// Get the current default route UID
	//
	uid, e := svs.protocol.GetDefaultRouteUID()
	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Default Route error:%s", e)
		return e
	}

	//
	// Create Submit if validator is running
	//

	// Is validator running?

	// Lots of error checking needed here, or a better way of pulling out parameters in a controlled manner
	username := svs.dstDest[uid].Username()
	minerID := svs.minerRec.ID
	destID := svs.minerRec.Dest
	jobID := request.Params[1].(string)
	extranonce := request.Params[2].(string)
	ntime := request.Params[3].(string)
	nonce := request.Params[4].(string)

	cs := contextlib.GetContextStruct(svs.Ctx())
	ps := cs.GetMsgBus()
	ps.SendValidateSubmit(svs.Ctx(), username, minerID, destID, jobID, extranonce, ntime, nonce)

	//
	// Get the username of the default route
	//

	// msg, e := request.createRequestMsg()
	msg, e := request.createSubmitRequestMsg(username)
	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" createRequestMsg() error:%s", e)
	}

	// Write to the current destination

	LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_SRC2DST, msg)

	count, e := svs.protocol.Write(msg)
	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" WriteSrc error:%s", e)
		return e
	}
	if count != len(msg) {
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" WriteSrc bad count:%d, %d", count, len(msg))
		e = fmt.Errorf(lumerinlib.FileLineFunc()+" WriteSrc bad count:%d, %d", count, len(msg))
		return e
	}

	// Call switchDest to change destinations if needed and we are set for OnSubmit
	if svs.scheduler == OnSubmit {
		svs.switchDest()
	}

	return nil
}

//
//
//
func (svs *StratumV1Struct) handleSrcReqCapabilities(request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" this is not handled yet")
	return ErrSrcReqNotSupported
}

//
//
//
func (svs *StratumV1Struct) handleSrcReqDifficulty(request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" this is not handled yet")
	return ErrSrcReqNotSupported

}

//
//
//
func (svs *StratumV1Struct) handleSrcReqExtranonce(request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" enter")

	state := svs.GetSrcState()
	// Validate the current sstate of the SRC connection
	switch state {
	case SrcStateNew:
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Got Configure")
		return ErrBadSrcState
	case SrcStateSubscribed:
	case SrcStateAuthorized:
	case SrcStateRunning:
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Src state:%s", state)
	}

	//
	// Get the current default route UID
	//
	uid, _ := svs.protocol.GetDefaultRouteUID()
	if uid < 0 {
		// Need to store this for later use
		msg, e := request.createRequestMsg()
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" createRequestMsg error:%s", e)
			return e
		}

		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_STOR_SRC, msg)

		svs.srcConfigure = request

		response := &stratumResponse{
			ID:     request.ID,
			Error:  nil,
			Result: nil,
			Reject: nil,
		}

		respmsg, e := response.createSrcExtranonceResponseMsg()
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" createResponsMsg() error:%s", e)
			return e
		}

		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_STOR2SRC, respmsg)

		count, e := svs.protocol.WriteSrc(respmsg)
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Write error:%s", e)
			return e
		}
		if count != len(respmsg) {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Write bad count:%d, %d", count, len(msg))
			e = fmt.Errorf(lumerinlib.FileLineFunc()+" WriteSrc bad count:%d, %d", count, len(msg))
			return e
		}

		r := *request
		svs.srcConfigure = &r

		return e
	} else {

		dstID := contextlib.GetDest(svs.Ctx())
		if dstID == nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" GetDest() returned nil")
		}

		msg, e := request.createRequestMsg()
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" createRequestMsg error:%s", e)
			return e
		}

		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_SRC2DST, msg)
		count, e := svs.protocol.Write(msg)
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Write error:%s", e)
			return e
		}
		if count != len(msg) {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Write bad count:%d, %d", count, len(msg))
			e = fmt.Errorf(lumerinlib.FileLineFunc()+" WriteSrc bad count:%d, %d", count, len(msg))
			return e
		}

	}
	return e

}

//
//
//
func (svs *StratumV1Struct) handleSrcReqSuggestTarget(request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" this is not handled yet")
	return ErrSrcReqNotSupported

}

// -------------------------------------------------------------------------
// Incoming Comm Dst REQUESTs
// -------------------------------------------------------------------------

//
// handleDstReqNotify()
// if the state of the connection is accetable, and the connection is the default pass the notification
//
//
func (svs *StratumV1Struct) handleDstReqNotify(uid simple.ConnUniqueID, request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" enter")

	msg, e := request.createReqMiningNotify()

	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" createResponseMsg() error:%s", e)
	} else {

		// is uid the current default destination?
		// If not, store the notify?
		// If so, pass it to the Src
		dststate := svs.GetDstStateUid(uid)
		switch dststate {
		case DstStateSubscribing:
			fallthrough
		case DstStateAuthorizing:
			fallthrough
		case DstStateStandBy:
			contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" State:%s", dststate)

			svs.dstLastReqNotify[uid] = request
			LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_STOR_DST, msg)
			return nil

		case DstStateRunning:
			// Record the last notify in case it is recalled after a switch dest call
			contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" passing notify for state:%s", dststate)

			minerID := svs.minerRec.ID
			username := svs.dstDest[uid].Username()
			destID := svs.minerRec.Dest
			n := request.Params
			jobID := n[0].(string)
			prevblock := n[1].(string)
			gen1 := n[2].(string)
			gen2 := n[3].(string)
			merkel := n[4].([]interface{})
			version := n[5].(string)
			nbits := n[6].(string)
			ntime := n[7].(string)
			clean := n[8].(bool)

			cs := contextlib.GetContextStruct(svs.Ctx())
			ps := cs.GetMsgBus()
			ps.SendValidateNotify(svs.Ctx(), minerID, destID, username, jobID, prevblock, gen1, gen2, merkel, version, nbits, ntime, clean)

			LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_DST2SRC, msg)
			svs.dstLastReqNotify[uid] = request
			svs.protocol.WriteSrc(msg)

		// case DstStateError:
		// case DstStateNew:
		default:
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Dst ReqNotify state:%s not handled", dststate)
			LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_DROP_DST, msg)
			return ErrDstReqNotSupported
		}
	}

	return e
}

//
//
//
func (svs *StratumV1Struct) handleDstReqGetVersion(UID simple.ConnUniqueID, request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" this is not handled yet")
	return ErrDstReqNotSupported

}

//
//
//
func (svs *StratumV1Struct) handleDstReqPing(UID simple.ConnUniqueID, request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" this is not handled yet")
	return ErrDstReqNotSupported

}

//
//
//
func (svs *StratumV1Struct) handleDstReqShowMessage(UID simple.ConnUniqueID, request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" this is not handled yet")
	return ErrDstReqNotSupported

}

//
//
//
func (svs *StratumV1Struct) handleDstReqSetExtranonce(UID simple.ConnUniqueID, request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" this is not handled yet")
	return ErrDstReqNotSupported

}

//
//
//
func (svs *StratumV1Struct) handleDstReqSetGoal(UID simple.ConnUniqueID, request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" this is not handled yet")
	return ErrDstReqNotSupported

}

//
// handleDstReqSetDifficulty()
// handles incomin set difficulty message from a pool connection
//
func (svs *StratumV1Struct) handleDstReqSetDifficulty(uid simple.ConnUniqueID, request *stratumRequest) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" enter")

	dststate := svs.GetDstStateUid(uid)
	switch dststate {
	case DstStateSubscribing:
		fallthrough
	case DstStateAuthorizing:
		fallthrough
	case DstStateRunning:
		fallthrough
	case DstStateStandBy:
		contextlib.Logf(svs.Ctx(), contextlib.LevelDebug, lumerinlib.FileLineFunc()+" Save set_diff on state :%s", dststate)
		msg, e := request.createRequestMsg()
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" createResponseMsg() error:%s", e)
		}
		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_STOR_DST, msg)

		svs.setLastReqSetDifficulty(uid, request)

	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Dst ReqNotify state:%s not handled", dststate)
		return ErrDstReqNotSupported
	}

	defRouteUid, _ := svs.protocol.GetDefaultRouteUID()
	// This is the default route
	if defRouteUid == uid {
		diff, e := request.getSetDifficulty()
		if e != nil {
			return e
		}

		cs := contextlib.GetContextStruct(svs.Ctx())
		ps := cs.GetMsgBus()
		ps.SendValidateSetDiff(svs.Ctx(), svs.minerRec.ID, svs.dstDest[uid].ID, diff)

		msg, e := request.createRequestSetDifficultyMsg()
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" createNoticeSetDifficultyMsg() returned error:%s", e)
			return e
		}

		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_DST2SRC, msg)

		svs.protocol.WriteSrc(msg)
	} else {
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" uid:%d is not the default dst:%d, should we store or drop the message", uid, defRouteUid)
	}

	return e
}

// -------------------------------------------------------------------------
// Incoming Comm NOTICEs
// -------------------------------------------------------------------------

//
// handleDstNotify()
// if the state of the connection is accetable, and the connection is the default pass the notification
//
//
func (svs *StratumV1Struct) handleDstNoticeNotify(uid simple.ConnUniqueID, notice *stratumNotice) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" enter")

	msg, e := notice.createNoticeMiningNotify()

	// is uid the current default destination?
	// If not, store the notify?
	// If so, pass it to the Src
	dststate := svs.GetDstStateUid(uid)
	switch dststate {
	case DstStateAuthorizing:
		fallthrough
	case DstStateSubscribing:
		fallthrough
	case DstStateRedialing:
		fallthrough
	case DstStateStandBy:
		contextlib.Logf(svs.Ctx(), contextlib.LevelWarn, lumerinlib.FileLineFunc()+" State:%s... Store the message", dststate)
		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_STOR_DST, msg)

	case DstStateRunning:
		defRouteUid, _ := svs.protocol.GetDefaultRouteUID()

		if defRouteUid == uid {
			minerID := svs.minerRec.ID
			destID := svs.minerRec.Dest
			username := svs.dstDest[uid].Username()
			n := notice.Params.([]interface{})
			jobID := n[0].(string)
			prevblock := n[1].(string)
			gen1 := n[2].(string)
			gen2 := n[3].(string)
			merkel := n[4].([]interface{})
			version := n[5].(string)
			nbits := n[6].(string)
			ntime := n[7].(string)
			clean := n[8].(bool)

			cs := contextlib.GetContextStruct(svs.Ctx())
			ps := cs.GetMsgBus()
			ps.SendValidateNotify(svs.Ctx(), minerID, destID, username, jobID, prevblock, gen1, gen2, merkel, version, nbits, ntime, clean)

			LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_DST2SRC, msg)
			svs.protocol.WriteSrc(msg)
			if e != nil {
				contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" createNoticeMiningNotify() returned error:%s", e)
				return e
			}
		} else {
			contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" UID[%d] is running but not default UID[%d]", uid, defRouteUid)
		}

	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+"  state:%s not handled", dststate)
	}

	e = notice.setNoticeMiningNotifyCleanJobsTrue()
	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" error:%s", e)
	}

	e = svs.setLastMiningNotice(uid, notice)
	if e != nil {
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" error:%s", e)
	}

	return e
}

//
// handleDstSetNoticeDifficulty()
// handles incomin set difficulty message from a pool connection
//
func (svs *StratumV1Struct) handleDstNoticeSetDifficulty(uid simple.ConnUniqueID, notice *stratumNotice) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" enter")

	// is index the current default destination?
	// If not, store the notify?
	// If so, pass it to the Src

	dststate := svs.GetDstStateUid(uid)
	switch dststate {
	case DstStateSubscribing:
		fallthrough
	case DstStateAuthorizing:
		fallthrough
	case DstStateStandBy:
		contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" storing difficulty for state:%s", dststate)

		e = svs.setLastSetDifficultyNotice(uid, notice)
		return nil

	case DstStateRunning:
		contextlib.Logf(svs.Ctx(), contextlib.LevelInfo, lumerinlib.FileLineFunc()+" passing set diff for state:%s", dststate)

		e = svs.setLastSetDifficultyNotice(uid, notice)
		if e != nil {
			return e
		}

	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+"  state:%s not handled", dststate)
	}

	//
	// If Default Route not set, set it.
	//
	defRouteUid, _ := svs.protocol.GetDefaultRouteUID()
	if defRouteUid < 0 {
		e = svs.protocol.SetDefaultRouteUID(uid)
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" SetDefaultRouteUID() error:%s", e)
			return e
		}

		defRouteUid = uid
	}

	// This is the default route
	if defRouteUid == uid {

		diff, e := notice.getSetDifficulty()
		if e != nil {
			return e
		}

		cs := contextlib.GetContextStruct(svs.Ctx())
		ps := cs.GetMsgBus()
		ps.SendValidateSetDiff(svs.Ctx(), svs.minerRec.ID, svs.dstDest[uid].ID, diff)

		msg, e := notice.createNoticeSetDifficultyMsg()
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" createNoticeSetDifficultyMsg() returned error:%s", e)
			return e
		}

		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_DST2SRC, msg)

		svs.protocol.WriteSrc(msg)
	} else {
		// Store or drop the message?
		contextlib.Logf(svs.Ctx(), contextlib.LevelInfo, lumerinlib.FileLineFunc()+" uid:%d is not the default dst:%d, should we store or drop the message", uid, defRouteUid)
	}

	return e
}

//
// handleDstSetNoticeExtranonce()
// handles incomin set difficulty message from a pool connection
//
func (svs *StratumV1Struct) handleDstNoticeSetExtranonce(uid simple.ConnUniqueID, notice *stratumNotice) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" enter")

	// is index the current default destination?
	// If not, store the notify?
	// If so, pass it to the Src

	e1, e2size, e := notice.getSetExtranonce()
	if e != nil {
		return e
	}

	svs.dstExtranonce[uid] = e1
	svs.dstExtranonce2size[uid] = e2size

	dststate := svs.GetDstStateUid(uid)
	switch dststate {
	case DstStateSubscribing:
		fallthrough
	case DstStateAuthorizing:
		fallthrough
	case DstStateStandBy:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" Stor the Extranonce data for State:%s", dststate)
		return nil

	case DstStateRunning:
		contextlib.Logf(svs.Ctx(), contextlib.LevelInfo, lumerinlib.FileLineFunc()+" passing set diff for state:%s", dststate)

	case DstStateError:
		fallthrough
	case DstStateNew:
		fallthrough
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+"  state:%s not handled", dststate)
	}

	//
	// If Default Route not set, set it.
	//
	defRouteUid, _ := svs.protocol.GetDefaultRouteUID()
	if defRouteUid < 0 {
		e = svs.protocol.SetDefaultRouteUID(uid)
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" SetDefaultRouteUID() error:%s", e)
			return e
		}

		defRouteUid = uid
	}

	// This is the default route
	if defRouteUid == uid {

		msg, e := notice.createNoticeMsg()
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" createNoticeSetDifficultyMsg() returned error:%s", e)
			return e
		}

		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_DST2SRC, msg)

		svs.protocol.WriteSrc(msg)
	} else {
		// Store or drop the message?
		contextlib.Logf(svs.Ctx(), contextlib.LevelInfo, lumerinlib.FileLineFunc()+" uid:%d is not the default dst:%d, should we store or drop the message", uid, defRouteUid)
	}

	return e
}

//
//
//
func (svs *StratumV1Struct) handleDstNoticeReconnect(uid simple.ConnUniqueID, notice *stratumNotice) (e error) {

	contextlib.Logf(svs.Ctx(), contextlib.LevelTrace, lumerinlib.FileLineFunc()+" enter")

	// is index the current default destination?
	// If not, store the notify?
	// If so, pass it to the Src

	dststate := svs.GetDstStateUid(uid)
	switch dststate {
	case DstStateSubscribing:
		fallthrough
	case DstStateAuthorizing:
		fallthrough
	case DstStateRunning:
		contextlib.Logf(svs.Ctx(), contextlib.LevelInfo, lumerinlib.FileLineFunc()+" passing set diff for state:%s", dststate)
	case DstStateStandBy:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+" state not handled yet:%s", dststate)
	// case DstStateError:
	// case DstStateNew:
	case DstStateClosed:
		contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" Connecton is Marked closed, ignore reopen")
		return fmt.Errorf("connection is marked closed, cant reopen")
	default:
		contextlib.Logf(svs.Ctx(), contextlib.LevelPanic, lumerinlib.FileLineFunc()+"  state:%s not handled", dststate)
	}

	defRouteUid, _ := svs.protocol.GetDefaultRouteUID()

	// This is the default route
	if defRouteUid == uid {
		// msg, e := notice.createNoticeSetDifficultyMsg()
		msg, e := notice.createNoticeMsg()
		if e != nil {
			contextlib.Logf(svs.Ctx(), contextlib.LevelError, lumerinlib.FileLineFunc()+" createNoticeSetDifficultyMsg() returned error:%s", e)
			return e
		}

		LogJson(svs.Ctx(), lumerinlib.FileLineFunc(), JSON_SEND_DST2SRC, msg)

		svs.protocol.WriteSrc(msg)
	} else {
		// Store or drop the message?
		contextlib.Logf(svs.Ctx(), contextlib.LevelInfo, lumerinlib.FileLineFunc()+" uid:%d is not the default dst:%d, should we store or drop the message", uid, defRouteUid)
	}

	return e

}
