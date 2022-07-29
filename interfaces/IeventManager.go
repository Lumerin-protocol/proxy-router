package interfaces

import (
	"context"
)

type IEventManager interface {

	// Attaches a subscriber to an event
	Subscribe(ctx context.Context, eventName string, onMessage func(interface{}))

	// Publishes the event across all the subscribers
	Publish(eventName string, data interface{})
}
