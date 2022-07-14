package interfaces

type IConnectionController interface {
	CreateNewConnection(id string, minerAddress string, destinationAddress string, status string) IConnectionModel
	AddConnection(id string, minerAddress string, destinationAddress string, status string) (IConnectionModel, error)
	RemoveConnection(id string) error
	GetConnection(id string) IConnectionModel
	GetConnections() ([]IConnectionModel, error)
}
