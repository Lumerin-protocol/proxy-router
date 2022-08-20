package lib

import (
	"fmt"
	"net/url"

	"gitlab.com/TitanInd/hashrouter/interfaces"
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

func BuildDestUri(scheme string, address string, user string, password string) string {
	userInfo := url.UserPassword(user, password)

	return fmt.Sprintf("%v://%v%v", scheme, userInfo.String(), address)
}

func BuildDest(scheme string, address string, user string, password string) (*Dest, error) {
	uri := BuildDestUri(scheme, address, user, password)

	return ParseDest(uri)
}

func (v *Dest) GetHost() string {
	return v.Host
}

func (v *Dest) Username() string {
	return v.User.Username()
}

func (v *Dest) Password() string {
	pwd, _ := v.User.Password()
	return pwd
}

func (v *Dest) IsEqual(target interfaces.IDestination) bool {
	return v.String() == target.String()
}

//compile time interfaces compatibility check
var _ interfaces.IDestination = (*Dest)(nil)

