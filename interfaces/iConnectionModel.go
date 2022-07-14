package interfaces

type IConnectionModel interface {
	GetId() string
	GetMinerAddress() string
	GetPoolAddress() string
	AddressesMatch(address string, destination string) bool
	SetAvailable(bool)
}
