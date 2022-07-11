package eventbus

import (
	"reflect"
	"testing"
)

func TestController(t *testing.T) {
	eb := NewEventBus()
	c := NewEventBusController(&eb)
	data := ConnectionEventData{Addr: "0.0.0.0", Name: "Test"}

	ch := make(chan ConnectionEventData)

	c.SubscribeConnection(ch)
	c.PublishConnection(data)

	t.Log("Wait")

	val := <-ch
	if !reflect.DeepEqual(data, val) {
		t.Fail()
	}
}
