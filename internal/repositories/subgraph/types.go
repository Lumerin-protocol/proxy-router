package subgraph

type Position struct {
	DeliveryAt string
	DestURL    string `graphql:"destURL"`
	ID         string
	IsPaid     bool
	Seller     struct {
		Address string
	}
	Buyer struct {
		Address string
	}
}
