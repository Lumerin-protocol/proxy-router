package interfaces

type IDestination interface {
	Username() string
	Password() string
	String() string
	GetHost() string
}
