package data

import (
	"github.com/stretchr/testify/require"
	"testing"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type TestModel struct {
}

func (*TestModel) GetID() string {
	return "testid"
}

func (t *TestModel) SetID(id string) interfaces.IBaseModel {
	return t
}

func (t *TestModel) Test() error {
	return nil
}

type ITestModel interface {
	interfaces.IBaseModel
	Test() error
}

var transactionsChannel chan func()

func TestNewInMemoryRepository(t *testing.T) {
	transactionsChannel = NewTransactionsChannel()

	repo := NewInMemoryRepository[ITestModel](nil, NewInMemoryDataStore(), transactionsChannel)
	model, err := repo.Create(&TestModel{})
	require.Nil(t, err)

	err = model.Test()
	require.Nil(t, err)

	id := model.GetID()
	require.Equal(t, id, "testid")

	_, err = repo.Save(model)
	require.Nil(t, err)

	_, err = repo.Get("testid213")
	require.NotNil(t, err)

	item, err := repo.Update(model)
	require.Nil(t, err)
	require.Equal(t, item, model)

	item, err = repo.Delete(model)
	require.Nil(t, err)
	require.Equal(t, item, nil)

	_, err = repo.FindOne(model)
	require.Nil(t, err)

	// TODO: Update with better scenario
	items := repo.Query(model)
	require.Nil(t, items)
}
