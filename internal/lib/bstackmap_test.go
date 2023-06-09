package lib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBstackMapCount(t *testing.T) {
	bsm := makeSampleBSM()
	assert.Equal(t, 3, bsm.Count())
}

func TestBstackMapCapacity(t *testing.T) {
	bsm := makeSampleBSM()
	bsm.Push("fourth", 4)
	assert.Equal(t, 3, bsm.Count())
	assert.Equal(t, 3, bsm.Capacity())
}

func TestBstackMapOverwrite(t *testing.T) {
	bsm := makeSampleBSM()
	bsm.Push("fourth", 4)
	_, ok := bsm.Get("first")
	assert.False(t, ok)
}

func TestBstackMapAt(t *testing.T) {
	bsm := makeSampleBSM()
	item, _ := bsm.At(0)
	assert.Equal(t, 1, item)

	item, _ = bsm.At(-1)
	assert.Equal(t, 3, item)
}

func TestBstackMapClear(t *testing.T) {
	bsm := makeSampleBSM()
	bsm.Clear()
	assert.Equal(t, 0, bsm.Count())
	assert.Equal(t, 3, bsm.Capacity())

	_, ok := bsm.Get("second")
	assert.False(t, ok)

	_, ok = bsm.At(0)
	assert.False(t, ok)
}

func makeSampleBSM() *BoundStackMap[int] {
	bsm := NewBoundStackMap[int](3)
	bsm.Push("first", 1)
	bsm.Push("second", 2)
	bsm.Push("third", 3)
	return bsm
}

func TestKK(t *testing.T) {
	bsm := NewBoundStackMap[interface{}](10)
	bsm.Push("119cf47985", 0)
	bsm.Push("119ce6bdab", 0)
	bsm.Push("119cefe562", 0)
	bsm.Push("119cfda164", 0)
	bsm.Push("119cf90e06", 0)
	bsm.Push("119ceb520c", 0)
	d, ok := bsm.Get("119cfda164")
	fmt.Println(d)
	fmt.Println(ok)
}
