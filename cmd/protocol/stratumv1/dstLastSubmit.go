package stratumv1

import (
	"context"
	"sync"
	"time"

	simple "gitlab.com/TitanInd/lumerin/cmd/lumerinnetwork/SIMPL"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

const requestTimeOutSec = 30

//
// Used to manage submits to a pool
// When an ACK response is recieved send the submit to the pool
// If a NAK is received delete it
// Periodically scan the request map for stale entries and remove
// so they do not accumulate over time
type stratumDstLastSubmitStruct struct {
	ctx     context.Context
	m       sync.Mutex
	request map[simple.ConnUniqueID]map[int]*stratumRequest
}

//
// newDstLastSubmit()
//
func newDstLastSubmit(cx context.Context) (dls *stratumDstLastSubmitStruct) {
	dls = &stratumDstLastSubmitStruct{
		ctx:     cx,
		request: make(map[simple.ConnUniqueID]map[int]*stratumRequest),
	}
	return dls
}

func (dls *stratumDstLastSubmitStruct) lock()   { dls.m.Lock() }
func (dls *stratumDstLastSubmitStruct) unlock() { dls.m.Unlock() }

//
// AddRequest()
// Insert a request into the LastSubmit structure
//
func (dls *stratumDstLastSubmitStruct) AddRequest(uid simple.ConnUniqueID, id int, r *stratumRequest) (e error) {
	dls.lock()
	defer dls.unlock()

	if dls.request[uid][id] != nil {
		contextlib.Logf(dls.ctx, contextlib.LevelPanic, lumerinlib.FileLineFunc()+" overwriting Last Submit record UID:%d, ID:%d", uid, id)
	}

	if dls.request[uid] == nil {
		dls.request[uid] = make(map[int]*stratumRequest)
	}
	dls.request[uid][id] = r

	go dls.goRequestTimeout(uid, id)
	// add timeout here...

	return e
}

//
// GetAndRemoveRequest()
// Find, remove, and return submit request associated with a DST and the request ID
//
func (dls *stratumDstLastSubmitStruct) GetAndRemoveRequest(uid simple.ConnUniqueID, id int) (r *stratumRequest, e error) {

	dls.lock()
	defer dls.unlock()

	if dls.request[uid] != nil {
		if dls.request[uid][id] != nil {
			r = dls.request[uid][id]
			delete(dls.request[uid], id)
			return r, e
		}
	}

	return nil, e
}

//
// goRequestTimeout()
// removes Submit requests after timeout period to prevent build up
//
func (dls *stratumDstLastSubmitStruct) goRequestTimeout(uid simple.ConnUniqueID, id int) {

	select {
	case <-dls.ctx.Done():

	case <-time.After(time.Second * requestTimeOutSec):

		dls.lock()
		defer dls.unlock()

		if dls.request[uid] != nil {
			if dls.request[uid][id] != nil {
				delete(dls.request[uid], id)
				contextlib.Logf(dls.ctx, contextlib.LevelWarn, lumerinlib.FileLineFunc()+" Submit Request Timed out UID:%d, ID:%d", uid, id)
			}
		}
	}
}
