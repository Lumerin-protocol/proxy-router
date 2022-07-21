package interfaces

type IMiningRequestProcessor interface {
	// Take the mining message data and return the data back;
	// The return value may or may not include chagnes depending on business requirements.
	MinerMessageHandler(message []byte) []byte
	// Take the pool message data from the pool and return the data back;
	// The return value may or may not include chagnes depending on business requirements.
	PoolMessageHandler(message []byte) []byte
	SetAuth(user string, password string)
}
