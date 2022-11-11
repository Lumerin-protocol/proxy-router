package interfaces

type IConnectionPoolDialer interface {
	Listen(network string, address string) (IConnectionPoolListener, error)
}

type IConnectionPoolListener interface {
}
