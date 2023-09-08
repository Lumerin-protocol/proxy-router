package hashrate

import (
	"net/url"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
)

type BlockchainState int

const (
	BlockchainStateAvailable BlockchainState = 0
	BlockchainStateRunning   BlockchainState = 1
)

type Base struct {
	ContractID string
	Seller     string
	Buyer      string
	StartsAt   *time.Time
	Duration   time.Duration
	Hashrate   float64
	State      BlockchainState
}

func (b *Base) GetID() string {
	return b.ContractID
}

func (b *Base) GetSeller() string {
	return b.Seller
}

func (b *Base) GetBuyer() string {
	return b.Buyer
}

func (b *Base) GetStartsAt() *time.Time {
	return b.StartsAt
}

func (b *Base) GetDuration() time.Duration {
	return b.Duration
}

func (b *Base) GetHashrateGHS() float64 {
	return b.Hashrate
}

func (b *Base) GetState() BlockchainState {
	return b.State
}

type Terms struct {
	Base
	Dest *url.URL
}

func (t *Terms) Encrypt(privateKey string) (*Terms, error) {
	var destUrl *url.URL

	if t.Dest != nil {
		dest, err := lib.EncryptString(t.Dest.String(), privateKey)
		if err != nil {
			return nil, err
		}

		destUrl, err = url.Parse(dest)
		if err != nil {
			return nil, err
		}
	} else {
		destUrl = nil
	}

	return &Terms{
		Base: Base{
			ContractID: t.ContractID,
			Seller:     t.Seller,
			Buyer:      t.Buyer,
			StartsAt:   t.StartsAt,
			Duration:   t.Duration,
			Hashrate:   t.Hashrate,
		},
		Dest: destUrl,
	}, nil
}

type EncryptedTerms struct {
	Base
	DestEncrypted string
}

func (t *EncryptedTerms) Decrypt(privateKey string) (*Terms, error) {
	var destUrl *url.URL

	if t.DestEncrypted != "" {
		dest, err := lib.DecryptString(t.DestEncrypted, privateKey)
		if err != nil {
			return nil, err
		}

		destUrl, err = url.Parse(dest)
		if err != nil {
			return nil, err
		}
	} else {
		destUrl = nil
	}

	return &Terms{
		Base: Base{
			ContractID: t.ContractID,
			Seller:     t.Seller,
			Buyer:      t.Buyer,
			StartsAt:   t.StartsAt,
			Duration:   t.Duration,
			Hashrate:   t.Hashrate,
		},
		Dest: destUrl,
	}, nil
}
