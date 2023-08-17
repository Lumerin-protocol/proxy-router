package lib

import (
	"bytes"
	"fmt"
)

type BoundStackMap[T any] struct {
	capacity int
	data     []string
	dataMap  map[string]T
	cc       int // current capacity
}

func NewBoundStackMap[T any](size int) *BoundStackMap[T] {
	var bs = new(BoundStackMap[T])
	bs.capacity = size
	bs.data = make([]string, 0, size)
	bs.dataMap = make(map[string]T, size)
	return bs
}

func (bs *BoundStackMap[T]) Push(key string, item T) {
	if bs.cc == bs.capacity {
		delete(bs.dataMap, bs.data[0])
		bs.data = bs.data[1:]
	} else {
		bs.cc++
	}
	bs.data = append(bs.data, key)
	bs.dataMap[key] = item
}

func (bs *BoundStackMap[T]) Get(key string) (T, bool) {
	item, ok := bs.dataMap[key]
	return item, ok
}

func (bs *BoundStackMap[T]) At(index int) (T, bool) {
	// adjustment for negative index values to be counted from the end
	if index < 0 {
		index = bs.cc + index
	}
	// check if index is out of bounds
	if index < 0 || index > (bs.cc-1) {
		return *new(T), false
	}
	return bs.dataMap[bs.data[index]], true
}

func (bs *BoundStackMap[T]) Clear() {
	bs.cc = 0
	bs.data = make([]string, 0, bs.capacity)
	bs.dataMap = make(map[string]T)
}

func (bs *BoundStackMap[T]) Count() int {
	return bs.cc
}

func (bs *BoundStackMap[T]) Capacity() int {
	return bs.capacity
}

func (bs *BoundStackMap[T]) Keys() []string {
	return bs.data
}

func (bs *BoundStackMap[T]) String() string {
	b := new(bytes.Buffer)
	for index, key := range bs.data {
		fmt.Fprintf(b, "(%d) %s: %v\n", index, key, bs.dataMap[key])
	}
	return b.String()
}
