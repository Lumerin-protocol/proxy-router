package resources

import (
	"net/url"
	"time"
)

type ContractData struct {
	ContractID        string
	Seller            string
	Buyer             string
	Dest              *url.URL
	StartedAt         *time.Time
	Duration          time.Duration
	ContractRole      ContractRole
	ResourceType      ResourceType
	ResourceEstimates map[string]float64 // hashrate
}
