package hashrate

import (
	"fmt"
	"math/big"
	"net/url"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
)

var (
	ErrInvalidDestURL    = fmt.Errorf("invalid url")
	ErrCannotDecryptDest = fmt.Errorf("cannot decrypt")
)

// Terms holds the terms of the contract where destination is decrypted
type Terms struct {
	BaseTerms
	dest *url.URL
}

func (p *Terms) Dest() *url.URL {
	return lib.CopyURL(p.dest)
}

func (t *Terms) Encrypt(privateKey string) (*Terms, error) {
	var destUrl *url.URL

	if t.dest != nil {
		dest, err := lib.EncryptString(t.dest.String(), privateKey)
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
		BaseTerms: *t.Copy(),
		dest:      destUrl,
	}, nil
}

// EncryptedTerms holds the terms of the contract where destination is encrypted
type EncryptedTerms struct {
	BaseTerms
	DestEncrypted string
}

func NewTerms(contractID, seller, buyer string, startsAt time.Time, duration time.Duration, hashrate float64, price *big.Int, state BlockchainState, isDeleted bool, balance *big.Int, hasFutureTerms bool, version uint32, destEncrypted string) *EncryptedTerms {
	return &EncryptedTerms{
		BaseTerms: BaseTerms{
			contractID:     contractID,
			seller:         seller,
			buyer:          buyer,
			startsAt:       startsAt,
			duration:       duration,
			hashrate:       hashrate,
			price:          price,
			state:          state,
			isDeleted:      isDeleted,
			balance:        balance,
			hasFutureTerms: hasFutureTerms,
			version:        version,
		},
		DestEncrypted: destEncrypted,
	}
}

func (t *EncryptedTerms) Decrypt(privateKey string) (*Terms, error) {
	var destUrl *url.URL

	if t.DestEncrypted != "" {
		dest, err := lib.DecryptString(t.DestEncrypted, privateKey)
		if err != nil {
			return nil, lib.WrapError(ErrCannotDecryptDest, fmt.Errorf("%s: %s", err, t.DestEncrypted))
		}

		destUrl, err = url.Parse(dest)
		if err != nil {
			return nil, lib.WrapError(ErrInvalidDestURL, fmt.Errorf("%s: %s", err, dest))
		}
	} else {
		destUrl = nil
	}

	return &Terms{
		BaseTerms: *t.Copy(),
		dest:      destUrl,
	}, nil
}

// BaseTerms holds the terms of the contract with common methods for both encrypted and decrypted terms
type BaseTerms struct {
	contractID     string
	seller         string
	buyer          string
	startsAt       time.Time
	duration       time.Duration
	hashrate       float64
	price          *big.Int
	state          BlockchainState
	isDeleted      bool
	balance        *big.Int
	hasFutureTerms bool
	version        uint32
}

func (b *BaseTerms) ID() string {
	return b.contractID
}

func (b *BaseTerms) Seller() string {
	return b.seller
}

func (b *BaseTerms) Buyer() string {
	return b.buyer
}

func (b *BaseTerms) StartTime() time.Time {
	return b.startsAt
}

func (p *BaseTerms) EndTime() time.Time {
	if p.startsAt.IsZero() {
		return time.Time{}
	}
	endTime := p.startsAt.Add(p.duration)
	return endTime
}

func (p *BaseTerms) Elapsed() time.Duration {
	if p.startsAt.IsZero() {
		return 0
	}
	return time.Since(p.startsAt)
}

func (b *BaseTerms) Duration() time.Duration {
	return b.duration
}

func (b *BaseTerms) HashrateGHS() float64 {
	return b.hashrate
}

func (b *BaseTerms) Price() *big.Int {
	return new(big.Int).Set(b.price) // copy
}

// PriceLMR returns price in LMR without decimals
func (b *BaseTerms) PriceLMR() float64 {
	price, _ := lib.NewRat(b.Price(), big.NewInt(1e8)).Float64()
	return price
}

func (p *BaseTerms) BlockchainState() BlockchainState {
	return p.state
}

func (b *BaseTerms) IsDeleted() bool {
	return b.isDeleted
}

func (b *BaseTerms) Balance() *big.Int {
	return new(big.Int).Set(b.balance) // copy
}

func (b *BaseTerms) HasFutureTerms() bool {
	return b.hasFutureTerms
}

func (b *BaseTerms) Version() uint32 {
	return b.version
}

func (b *BaseTerms) SetState(state BlockchainState) {
	b.state = state
}

func (b *BaseTerms) Copy() *BaseTerms {
	return &BaseTerms{
		contractID:     b.ID(),
		seller:         b.Seller(),
		buyer:          b.Buyer(),
		startsAt:       b.StartTime(),
		duration:       b.Duration(),
		hashrate:       b.HashrateGHS(),
		state:          b.BlockchainState(),
		price:          b.Price(),
		isDeleted:      b.IsDeleted(),
		balance:        b.Balance(),
		hasFutureTerms: b.HasFutureTerms(),
		version:        b.Version(),
	}
}
