package interfaces

type IDestination interface {
	Username() string

	Password() string

	IsEqual(target IDestination) bool
	String() string
	GetHost() string
}
