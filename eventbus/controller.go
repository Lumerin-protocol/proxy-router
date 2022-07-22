package eventbus

import (
	"context"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type EventBusController struct {
	eventBus interfaces.IEventManager
}

const (
	ConnectionEventName = "connection"
	ContractEventName   = "contract"
	TestEventName       = "test"
)

type ConnectionEventData struct {
	Addr string
	Name string
}

type ContractEventData struct {
	ID       string
	DestAddr string
}

func NewEventBusController(e interfaces.IEventManager) *EventBusController {
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

func (c *EventBusController) SubscribeConnection(ctx context.Context, ch chan<- ConnectionEventData) {
	c.eventBus.Subscribe(ctx, ConnectionEventName, func(val interface{}) {
		ch <- val.(ConnectionEventData)
	})
}

func (c *EventBusController) SubscribeConnectionCb(ctx context.Context, cb func(data ConnectionEventData)) {
	c.eventBus.Subscribe(ctx, ConnectionEventName, func(val interface{}) {
		cb(val.(ConnectionEventData))
	})
}

func (c *EventBusController) SubscribeContract(ctx context.Context, ch chan<- ContractEventData) {
	c.eventBus.Subscribe(ctx, ContractEventName, func(val interface{}) {
		ch <- val.(ContractEventData)
	})
}

func (c *EventBusController) SubscribeContract2(ctx context.Context, size int) <-chan ContractEventData {
	ch := make(chan ContractEventData, size)
	c.eventBus.Subscribe(ctx, ContractEventName, func(val interface{}) {
		ch <- val.(ContractEventData)
	})
	return ch
}

func (c *EventBusController) SubscribeContractCb(ctx context.Context, cb func(data ConnectionEventData)) {
	c.eventBus.Subscribe(ctx, ConnectionEventName, func(val interface{}) {
		cb(val.(ConnectionEventData))
	})
}
