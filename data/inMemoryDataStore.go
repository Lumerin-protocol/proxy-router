package data

type Store = map[string]map[string]interface{}

type TransactionsChannel = chan func()

func NewTransactionsChannel() TransactionsChannel {

	transactionsChannel := make(TransactionsChannel)

	go func() {
		for f := range transactionsChannel {
			f()
		}
	}()

	return transactionsChannel
}

func NewInMemoryDataStore() Store {
	return make(Store)
}
