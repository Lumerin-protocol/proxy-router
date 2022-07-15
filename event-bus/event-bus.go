package eventbus

import (
	"context"
	"fmt"
	"sync"
)

var EventChannelBufferSize = 1000000

type OnMessageFunc = func(interface{})
type DataEvent = interface{}
type DataChannel = chan DataEvent
type EventChannel struct {
	ListenerChannels map[int]DataChannel
	LastChannelID    int
}

type eventBus struct {
	eventChannels map[string]EventChannel
	rm            sync.RWMutex
	bufferSize    int
}

func NewEventBus() eventBus {
	return eventBus{
		eventChannels: map[string]EventChannel{},
		bufferSize:    EventChannelBufferSize,
	}
}

func (e *eventBus) SetBufferSize(bufferSize int) {
	e.bufferSize = bufferSize
}

// Subscribe for events from the bus. Creates one goroutine per subscription
// returns cancel function to stop subscription
func (e *eventBus) Subscribe(ctx context.Context, eventName string, onMessage OnMessageFunc) {
	e.rm.Lock()
	defer e.rm.Unlock()

	_, found := e.eventChannels[eventName]
	if !found {
		e.eventChannels[eventName] = EventChannel{
			LastChannelID:    0,
			ListenerChannels: make(map[int]DataChannel),
		}
	}
	internalChannel := make(DataChannel, e.bufferSize)

	eventChannel := e.eventChannels[eventName]
	eventChannel.LastChannelID = eventChannel.LastChannelID + 1
	eventChannel.ListenerChannels[eventChannel.LastChannelID] = internalChannel

	e.eventChannels[eventName] = eventChannel
	go e.startSubscriber(ctx, internalChannel, onMessage)
}

func (e *eventBus) startSubscriber(ctx context.Context, sourceCh DataChannel, onMessage OnMessageFunc) {
SUB:
	for {
		select {
		case <-ctx.Done():
			break SUB
			// close(destCh)
		case val := <-sourceCh:
			onMessage(val)
		}
	}

	fmt.Printf("Subscriber cancelled\n")
}

// Publishes event to event bus. Non-blocking if buffer is large enough
func (e *eventBus) Publish(eventName string, data DataEvent) {
	e.rm.RLock()
	defer e.rm.RUnlock()

	chans, found := e.eventChannels[eventName]
	if !found {
		fmt.Printf("No consumers for an event: discarding publish\n")
		// log that no messages
		return
	}

	for ID, ch := range chans.ListenerChannels {
		select {
		case ch <- data:
			fmt.Printf("Published to chan %d\n", ID)
			continue
		default:
			fmt.Printf("Blocking publish to event bus queue (%s): increase buffer size\n", eventName)
		}
	}
}
