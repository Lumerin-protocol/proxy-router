package eventbus

import (
	"context"
)

type EventBusController struct {
	eventBus *eventBus
}

const (
	ConnectionEventName = "connection"
	ContractEventName   = "contract"
)

type ConnectionEventData struct {
	Addr string
	Name string
}

type ContractEventData struct {
	ID       string
	DestAddr string
}

func NewEventBusController(e *eventBus) *EventBusController {
	return &EventBusController{
		eventBus: e,
	}
}

func (c *EventBusController) PublishConnection(data ConnectionEventData) {
	c.eventBus.Publish(ConnectionEventName, data)
}

func (c *EventBusController) PublishContract(data ContractEventData) {
	c.eventBus.Publish(ContractEventName, data)
}

func (c *EventBusController) SubscribeConnection(ch chan<- ConnectionEventData) context.CancelFunc {
	return c.eventBus.Subscribe(ConnectionEventName, func(val interface{}) {
		ch <- val.(ConnectionEventData)
	})
}

func (c *EventBusController) SubscribeContract(ch chan<- ContractEventData) context.CancelFunc {
	return c.eventBus.Subscribe(ContractEventName, func(val interface{}) {
		ch <- val.(ContractEventData)
	})
}
