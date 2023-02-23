package lib

import (
	"fmt"
	"net/url"
)

type Dest struct {
	url url.URL
}

type Stringable interface {
	String() string
}

func ParseDest(uri string) (Dest, error) {
	res, err := url.Parse(uri)
	if err != nil {
		return Dest{}, err
	}
	res.Scheme = "" // drop stratum+tcp prefix to avoid comparison issues
	return Dest{*res}, nil
}

func MustParseDest(uri string) Dest {
	res, err := ParseDest(uri)
	if err != nil {
		panic(err)
	}
	return res
}

func NewDest(workerName string, pwd string, host string, port *int) *Dest {
	if port != nil {
		host = fmt.Sprintf("%s:%d", host, port)
	}

	return &Dest{
		url.URL{
			User: url.UserPassword(workerName, pwd),
			Host: host,
		},
	}
}

func (v Dest) Username() string {
	return v.url.User.Username()
}

func (v Dest) Password() string {
	pwd, _ := v.url.User.Password()
	return pwd
}

func (v Dest) GetHost() string {
	return v.url.Host
}

func (v Dest) String() string {
	return v.url.String()
}

func IsEqualDest(dest1, dest2 Stringable) bool {
	return dest1.String() == dest2.String()
}
