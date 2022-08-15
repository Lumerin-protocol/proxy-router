package data

import (
	"testing"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type TestModel struct {
}

func (*TestModel) GetId() string {
	return "testid"
}

func (t *TestModel) SetId(id string) interfaces.IBaseModel {
	return t
}

func (t *TestModel) Test() {}

type ITestModel interface {
	interfaces.IBaseModel
	Test()
}

var transactionsChannel chan func()

func TestNewInMemoryRepository(t *testing.T) {
	transactionsChannel = NewTransactionsChannel()

	repo := NewInMemoryRepository[ITestModel](nil, NewInMemoryDataStore(), transactionsChannel)

	repo.Create(&TestModel{})
}
