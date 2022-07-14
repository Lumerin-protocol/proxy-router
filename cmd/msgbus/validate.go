package msgbus

import (
	"context"
	"fmt"

	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

type ValidateID IDString

type Validate struct {
	ID      ValidateID // "Random ID value"
	MinerID string
	DestID  string
	Data    interface{}
}

type Submit struct {
	WorkerName string
	JobID      string
	Extraonce  string
	NTime      string
	NOnce      string
}

type Notify struct {
	JobID           string
	UserID          string
	PrevBlockHash   string
	GenTransaction1 string
	GenTransaction2 string
	MerkelBranches  []interface{}
	Version         string
	Nbits           string
	Ntime           string
	Clean           bool
}

type SetDifficulty struct {
	Diff int
}

func newValidate(id ValidateID, minerID string, destID string, data interface{}) (v *Validate) {

	v = &Validate{
		ID:      id,
		MinerID: minerID,
		DestID:  destID,
		Data:    data,
	}

	return v
}

func newSubmit(workername string, jobid string, extranonce string, ntime string, nonce string) (s *Submit) {

	s = &Submit{
		WorkerName: workername,
		JobID:      jobid,
		Extraonce:  extranonce,
		NTime:      ntime,
		NOnce:      nonce,
	}

	return s
}

func newNotify(jobid string, username string, prevblock string, gen1 string, gen2 string, merkel []interface{}, version string, nbits string, ntime string, clean bool) (n *Notify) {

	n = &Notify{

		JobID:           jobid,
		UserID:          username,
		PrevBlockHash:   prevblock,
		GenTransaction1: gen1,
		GenTransaction2: gen2,
		MerkelBranches:  merkel,
		Version:         version,
		Nbits:           nbits,
		Ntime:           ntime,
		Clean:           clean,
	}

	return n
}

func newSetDiff(diff int) (d *SetDifficulty) {

	d = &SetDifficulty{
		Diff: diff,
	}

	return d
}

//
//
//
func getValidateID() ValidateID {
	id := fmt.Sprintf("ValidateID:%d", <-SubmitCountChan)
	return ValidateID(id)
}

//
//
//
func (ps *PubSub) SendValidateSubmit(ctx context.Context, workername string, m MinerID, d DestID, jobID string, extranonce string, ntime string, nonce string) {

	submit := newSubmit(workername, jobID, extranonce, ntime, nonce)
	id := getValidateID()
	validate := newValidate(id, string(m), string(d), submit)

	_, e := ps.Pub(ValidateMsg, IDString(id), validate)
	if e != nil {
		panic(fmt.Sprintf(lumerinlib.FileLineFunc()+" Pub() error:%s", e))
	}
}

//
//
//
func (ps *PubSub) SendValidateNotify(ctx context.Context, m MinerID, d DestID, username string, jobid string, prevblock string, gen1 string, gen2 string, merkel []interface{}, version string, nbits string, ntime string, clean bool) {

	notify := newNotify(jobid, username, prevblock, gen1, gen2, merkel, version, nbits, ntime, clean)
	id := getValidateID()
	validate := newValidate(id, string(m), string(d), notify)

	_, e := ps.Pub(ValidateMsg, IDString(id), validate)
	if e != nil {
		panic(fmt.Sprintf(lumerinlib.FileLineFunc()+" Pub() error:%s", e))
	}
}

//
//
//
func (ps *PubSub) SendValidateSetDiff(ctx context.Context, m MinerID, d DestID, diff int) {

	setdiff := newSetDiff(diff)
	id := getValidateID()
	validate := newValidate(id, string(m), string(d), setdiff)

	_, e := ps.Pub(ValidateMsg, IDString(id), validate)
	if e != nil {
		panic(fmt.Sprintf(lumerinlib.FileLineFunc()+" Pub() error:%s", e))
	}
}
