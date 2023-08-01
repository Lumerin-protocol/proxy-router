package validator

import (
	"encoding/hex"

	sm "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/stratumv1_message"
)

type MiningJob struct {
	notify          *sm.MiningNotify
	diff            float64
	extraNonce1     string
	extraNonce2Size int
	shares          map[[24]byte]bool
}

func NewMiningJob(msg *sm.MiningNotify, diff float64, extraNonce1 string, extraNonce2Size int) *MiningJob {
	return &MiningJob{
		notify:          msg,
		diff:            diff,
		extraNonce1:     extraNonce1,
		extraNonce2Size: extraNonce2Size,
		shares:          make(map[[24]byte]bool, 32),
	}
}

func (m *MiningJob) CheckDuplicateAndAddShare(s *sm.MiningSubmit) bool {
	bytes := HashShare("00000000", s.GetExtraNonce2(), s.GetNtime(), s.GetNonce(), s.GetVmask())

	if m.shares[bytes] {
		return true
	}

	m.shares[bytes] = true
	return false
}

func (m *MiningJob) GetNotify() *sm.MiningNotify {
	return m.notify.Copy()
}

func (m *MiningJob) GetDiff() float64 {
	return m.diff
}

func (m *MiningJob) GetExtraNonce1() string {
	return m.extraNonce1
}

func (m *MiningJob) GetExtraNonce2Size() int {
	return m.extraNonce2Size
}

func HashShare(enonce1, enonce2, ntime, nonce, vmask string) [24]byte {
	var hash [24]byte

	enonce1Bytes, _ := hex.DecodeString(enonce1)
	enonce2Bytes, _ := hex.DecodeString(enonce2)
	ntimeBytes, _ := hex.DecodeString(ntime)
	nonceBytes, _ := hex.DecodeString(nonce)
	vmaskBytes, _ := hex.DecodeString(vmask)

	copy(hash[:4], enonce1Bytes[:4])
	copy(hash[4:12], enonce2Bytes[:8])
	copy(hash[12:16], ntimeBytes[:4])
	copy(hash[16:20], nonceBytes[:4])
	copy(hash[20:24], vmaskBytes[:4])

	return hash
}
