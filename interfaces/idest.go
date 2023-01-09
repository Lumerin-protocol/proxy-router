package interfaces

import "gitlab.com/TitanInd/hashrouter/lib"

type IDestination interface {
	Username() string
	Password() string
	IsEqual(target lib.DestString) bool
	String() string
	GetHost() string
}
