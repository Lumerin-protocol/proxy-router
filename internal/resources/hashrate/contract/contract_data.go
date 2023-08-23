package contract

import (
	"time"
)

type ContractData struct {
	ContractID     string
	Seller         string
	Buyer          string
	EncryptedDest  string
	HashrateGHS    int
	StartedAt      time.Time
	Duration       time.Duration
	HasFutureTerms bool
	IsDeleted      bool
	State          uint8
}
