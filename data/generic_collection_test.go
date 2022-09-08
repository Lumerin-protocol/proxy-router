package data

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
