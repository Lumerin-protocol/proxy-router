package eventbus

import (
	"context"
	"reflect"
	"testing"
)

func TestSubscribeConnection(t *testing.T) {
	eb := NewEventBus()
	c := NewEventBusController(eb)
	data := ConnectionEventData{Addr: "0.0.0.0", Name: "Test"}

	ch := make(chan ConnectionEventData)

	c.SubscribeConnection(context.Background(), ch)
	c.PublishConnection(data)

	t.Log("Wait")

	val := <-ch
	if !reflect.DeepEqual(data, val) {
		t.Fail()
	}
}

func TestSubscribeConnectionCb(t *testing.T) {
	eb := NewEventBus()
	c := NewEventBusController(eb)
	data := ConnectionEventData{Addr: "0.0.0.0", Name: "Test"}

	ch := make(chan ConnectionEventData)
	c.SubscribeConnectionCb(context.Background(), func(val ConnectionEventData) {
		ch <- val
	})
	c.PublishConnection(data)

	res := <-ch
	if !reflect.DeepEqual(data, res) {
		t.Fail()
	}
}
