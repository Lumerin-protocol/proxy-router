package interop

import (
	"net/url"
)

type Dest struct {
	url.URL
}

func ParseDest(uri string) (*Dest, error) {
	res, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	res.Scheme = "" // drop stratum+tcp prefix to avoid comparison issues
	return &Dest{*res}, nil
}

func (v Dest) Username() string {
	return v.User.Username()
}

func (v Dest) Password() string {
	pwd, _ := v.User.Password()
	return pwd
}

func (v Dest) IsEqual(target Dest) bool {
	return v.String() == target.String()
}
