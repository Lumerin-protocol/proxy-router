package eventbus

import (
	"context"
	"fmt"
	"sync"
)

var EventChannelBufferSize = 1000000

type CastFunc = func(interface{})
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
func (e *eventBus) Subscribe(eventName string, castFn CastFunc) context.CancelFunc {
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

	ctx, cancel := context.WithCancel(context.Background())
	go e.startSubscribePiping(ctx, internalChannel, castFn)

	return cancel
}

func (e *eventBus) startSubscribePiping(ctx context.Context, sourceCh DataChannel, castFn CastFunc) {
	for {
		select {
		case <-ctx.Done():
			// close(destCh)
			return
		case val := <-sourceCh:
			castFn(val)
		}
	}
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
