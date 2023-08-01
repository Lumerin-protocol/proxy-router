package validator

import (
	"errors"
	"fmt"

	gi "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	sm "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
)

type Destination interface {
	GetVersionRolling() (versionRolling bool, versionRollingMask string)
	GetExtraNonce() (extraNonce string, extraNonceSize int)
}

var (
	ErrJobNotFound    = errors.New("job not found")
	ErrDuplicateShare = errors.New("duplicate share")
	ErrLowDifficulty  = errors.New("low difficulty")
)

type Validator struct {
	// state
	jobs *lib.BoundStackMap[*miningJob]

	// deps
	dest Destination
	log  gi.ILogger
}

func NewValidator(dest Destination, log gi.ILogger) *Validator {
	return &Validator{
		jobs: lib.NewBoundStackMap[*miningJob](30),
		dest: dest,
		log:  log,
	}
}

func (v *Validator) AddNewJob(msg *sm.MiningNotify, diff float64) {
	job := NewMiningJob(msg, diff)
	v.jobs.Push(msg.GetJobID(), job)
}

func (v *Validator) ValidateAndAddShare(msg *sm.MiningSubmit) (float64, error) {
	var (
		job *miningJob
		ok  bool
	)

	if job, ok = v.jobs.Get(msg.GetJobId()); !ok {
		return 0, ErrJobNotFound
	}

	if job.CheckDuplicateAndAddShare(msg) {
		return 0, ErrDuplicateShare
	}

	_, mask := v.dest.GetVersionRolling()
	xn, xn2size := v.dest.GetExtraNonce()

	diff, ok := ValidateDiff(xn, uint(xn2size), uint64(job.diff), mask, job.notify, msg)
	diffFloat := float64(diff)
	if !ok {
		err := lib.WrapError(ErrLowDifficulty, fmt.Errorf("expected %.2f actual %d", job.diff, diff))
		v.log.Warnf(err.Error())
		return diffFloat, err
	}

	return diffFloat, nil
}

func (v *Validator) GetLatestJob() (*sm.MiningNotify, bool) {
	job, ok := v.jobs.At(-1)
	if !ok {
		return nil, false
	}
	return job.notify.Copy(), true
}
