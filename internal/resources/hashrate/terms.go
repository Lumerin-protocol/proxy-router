package hashrate

import (
	"net/url"
	"time"
)

type Terms struct {
	ContractID string
	Seller     string
	Buyer      string
	Dest       *url.URL
	StartedAt  *time.Time
	Duration   time.Duration
	Hashrate   float64
}
