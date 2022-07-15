package lib

import (
	"errors"
	"testing"
)

func TestWrapError(t *testing.T) {
	child := errors.New("child")
	parent := errors.New("parent")
	err3 := Wrap(parent, child)

	if !errors.Is(err3, parent) {
		t.Fail()
	}
	if !errors.Is(err3, child) {
		t.Fail()
	}
}
