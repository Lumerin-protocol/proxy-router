package data

import (
	"fmt"
	"reflect"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/interop"
)

type InMemoryRepository[T interfaces.IBaseModel] struct {
	transactionsChannel TransactionsChannel
	store               Store
	storageKey          string
	logger              interfaces.ILogger
}

func (repo *InMemoryRepository[T]) GetDefault() T {
	var defaultModel T
	return defaultModel
}

func (repo *InMemoryRepository[T]) MakeTransactionChannels() (chan T, chan error, func() (T, error)) {

	resultChannel := make(chan T)
	errorResultChannel := make(chan error)

	return resultChannel, errorResultChannel, func() (T, error) {
		select {
		case result := <-resultChannel:
			return result, nil
		case errorResult := <-errorResultChannel:
			return repo.GetDefault(), errorResult
		}
	}
}

// Deposit adds money to an account.
func (repo *InMemoryRepository[T]) Get(id string) (T, error) {

	resultChannel, errorResultChannel, callback := repo.MakeTransactionChannels()

	repo.transactionsChannel <- func() {
		item := repo.store[repo.storageKey][id]

		if item == nil {
			errorResultChannel <- fmt.Errorf("InMemoryRepository.Get - %v %v does not exist", repo.storageKey, id)
			return
		}

		resultItem, ok := item.(T)

		if ok {
			resultChannel <- resultItem
		}

		errorResultChannel <- fmt.Errorf("InMemoryRepository.Get - item is %v, not %v", reflect.TypeOf(item).String(), reflect.TypeOf(resultItem).String())
	}

	return callback()
}

func (repo *InMemoryRepository[T]) Save(payload T) (T, error) {

	if payload.GetId() != "" {
		return repo.Update(payload)
	}

	return repo.Create(payload)
}

func (repo *InMemoryRepository[T]) Create(payload T) (T, error) {
	id := interop.NewUniqueIdString()

	result := payload.SetId(id)

	repo.transactionsChannel <- func() {
		repo.store[repo.storageKey][id] = result
	}

	return result.(interface{}).(T), nil
}

func (repo *InMemoryRepository[T]) Update(payload T) (T, error) {
	id := payload.GetId()

	repo.transactionsChannel <- func() {
		repo.store[repo.storageKey][id] = payload
	}

	return payload, nil
}

func (repo *InMemoryRepository[T]) FindOne(payload T) (T, error) {
	return repo.Get(payload.GetId())
}

func (repo *InMemoryRepository[T]) Query(query T) []T {

	// return repo.db.Where(query).Value.([]T)

	return nil

}

func (repo *InMemoryRepository[T]) Delete(query T) (T, error) {
	// log.Printf("delete entity: %v", query)
	// repo.db.Delete(query)

	// return query, nil

	return repo.GetDefault(), nil
}

func NewInMemoryRepository[T interfaces.IBaseModel](logger interfaces.ILogger, store Store, transactionsChannel TransactionsChannel) interfaces.IRepository[T] {

	dataKey := reflect.TypeOf(new(T)).String()

	store[dataKey] = make(map[string]interface{})

	return &InMemoryRepository[T]{
		logger:              logger,
		store:               store,
		transactionsChannel: transactionsChannel,
		storageKey:          dataKey,
	}
}
