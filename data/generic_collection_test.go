package data

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type TestModel struct {
}

func (*TestModel) GetID() string {
	return "testid"
}
func (t *TestModel) SetID(id string) {
}
func (t *TestModel) Test() error {
	return nil
}

type ITestModel interface {
	GetID() string
	SetID(ID string)
	Test() error
}

func TestCollection(t *testing.T) {
	collection := NewCollection[ITestModel]()
	require.NotNil(t, collection)

	collection.Store(&TestModel{})

	item, ok := collection.Load("testid")
	require.Equal(t, ok, true)
	require.NotNil(t, item)

	collection.Delete("testid")

	item, ok = collection.Load("testid")
	require.Equal(t, ok, false)
	require.Nil(t, item)
}
