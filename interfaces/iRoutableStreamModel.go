package interfaces

type IRoutableStreamModel interface {
	GetID() string
	ChangeDestination(addr string, username string, password string) error
	GetCurrentDestination() string
}
