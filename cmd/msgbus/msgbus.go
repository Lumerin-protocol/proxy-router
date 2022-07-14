package msgbus

import (
	"crypto/rand"
	"fmt"

	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

type operation string
type EventType string
type MsgType string
type MsgBusError string

const (
	opNop      operation = "opNop"
	opPub      operation = "opPub"
	opSub      operation = "opSub"
	opGet      operation = "opGet"
	opSet      operation = "opSet"
	opSearch   operation = "opSearch"
	opUnsub    operation = "opUnsub"
	opUnpub    operation = "opUnpub"
	opRemove   operation = "opRemove"
	opShutdown operation = "opShutdown"
)

const (
	NoEvent           EventType = "NoEvent"
	UpdateEvent       EventType = "UpdEvent"
	DeleteEvent       EventType = "DelEvent"
	GetEvent          EventType = "GetEvent"
	GetIndexEvent     EventType = "GetIdxEvent"
	SearchEvent       EventType = "SearchEvent"
	SearchIndexEvent  EventType = "SearchIndexEvent"
	PublishEvent      EventType = "PubEvent"
	UnpublishEvent    EventType = "UnpubEvent"
	SubscribedEvent   EventType = "SubEvent"
	UnsubscribedEvent EventType = "UnsubEvent"
	RemovedEvent      EventType = "RemovedEvent"
)

const (
	NoMsg                    MsgType = "NoMsg"
	ConfigMsg                MsgType = "ConfigMsg"
	ContractManagerConfigMsg MsgType = "ContractManagerConfigMsg"
	DestMsg                  MsgType = "DestMsg"
	NodeOperatorMsg          MsgType = "NodeOperatorMsg"
	ContractMsg              MsgType = "ContractMsg"
	MinerMsg                 MsgType = "MinerMsg"
	ConnectionMsg            MsgType = "ConnectionMsg"
	LogMsg                   MsgType = "LogMsg"
	ValidateMsg              MsgType = "ValidateMsg"
)

type Event struct {
	EventType EventType
	Msg       MsgType
	ID        IDString
	RequestID int
	Data      interface{}
	Err       error
}

type EventChan chan *Event

type Subscribers struct {
	eventchan map[EventChan]int
}

type registryData struct {
	sub  Subscribers
	data interface{}
}

type IDIndex []IDString

type registry struct {
	data   map[MsgType]map[IDString]registryData
	notify map[MsgType]map[chan *Event]interface{}
}

// PubSub is a collection of topics.
type PubSub struct {
	cmdChan  chan *cmd
	capacity int
	// requestIDChan carries the incrementing request IDs
	requestIDChan chan int
	// done signals to close the requestIDChan
	done   chan struct{}
	logger *log.Logger
}

const (
	MsgBusErrNoErr         MsgBusError = "NoErr"
	MsgBusErrNoMsg         MsgBusError = "NoMsg"
	MsgBusErrNoID          MsgBusError = "NoID"
	MsgBusErrNoSub         MsgBusError = "NoSub"
	MsgBusErrNoData        MsgBusError = "NoData"
	MsgBusErrNoSearchTerm  MsgBusError = "NoSearchTerm"
	MsgBusErrNoEventChan   MsgBusError = "NoEventChan"
	MsgBusErrBadMsg        MsgBusError = "BadMsg"
	MsgBusErrBadID         MsgBusError = "BadID"
	MsgBusErrBadData       MsgBusError = "BadData"
	MsgBusErrBadSearchTerm MsgBusError = "BadSearchTerm"
	MsgBusErrDupID         MsgBusError = "DupID"
	MsgBusErrDupData       MsgBusError = "DupData"
)

type cmd struct {
	op operation
	// sync indicates how the response should be sent back to the caller,
	// synchronously (direct return) or asynchronously via a supplied channel.
	sync bool
	msg  MsgType
	ID   IDString
	// requestID is an incrementing value to keep track of each async call.
	requestID int
	Name      string
	IP        string
	MAC       string
	data      interface{}
	eventch   EventChan
	returnch  EventChan
}

var SubmitCountChan chan int

//
// init()
// initializes the DstCounter
//
func init() {
	SubmitCountChan = make(chan int, 5)
	lumerinlib.RunGoCounter(SubmitCountChan)
}

//--------------------------------------------------------------------------------
//
//--------------------------------------------------------------------------------
func getCommandError(e MsgBusError) error {
	return fmt.Errorf("command Error: %s", e)
}

// New creates a new PubSub and starts a goroutine for handling operations.
// The capacity of the channels created by Sub and SubOnce will be as specified.
func New(capacity int, l *log.Logger) *PubSub {
	ps := &PubSub{
		cmdChan:       make(chan *cmd),
		capacity:      capacity,
		requestIDChan: make(chan int),
		logger:        l,
	}
	go ps.start()

	return ps
}

// NewEventChan creates a new event channel for passing events.
func NewEventChan() EventChan {
	return make(EventChan)
}

// GetRandomIDString returns a random string.
// Format: xxxxxxxx-xxxxxxxx-xxxxxxxx-xxxxxxxx
func GetRandomIDString() (i IDString) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		fmt.Printf("Error reading random file: %s\n", err)
		panic(err)
	}
	str := fmt.Sprintf("%08x-%08x-%08x-%08x", b[0:4], b[4:8], b[8:12], b[12:16])
	i = IDString(str)
	return i
}

// Pub publishes a message/command to its subscribers, asynchronously.
func (ps *PubSub) Pub(msg MsgType, id IDString, data interface{}, ech ...EventChan) (requestID int, err error) {
	requestID = <-ps.requestIDChan

	if msg == NoMsg {
		return requestID, getCommandError(MsgBusErrNoMsg)
	}

	if id == "" {
		return requestID, getCommandError(MsgBusErrNoID)
	}

	if data == nil {
		return requestID, getCommandError(MsgBusErrNoData)
	}

	var eventchan EventChan = nil

	for _, v := range ech {
		if v != nil {
			eventchan = v
			break
		}
	}

	c := cmd{
		op:        opPub,
		sync:      false,
		msg:       msg,
		ID:        id,
		requestID: requestID,
		data:      data,
		eventch:   eventchan,
	}

	_, err = ps.dispatch(&c)

	return requestID, err
}

// PubWait publishes a message/command to its subscribers, synchronously.
func (ps *PubSub) PubWait(msg MsgType, id IDString, data interface{}) (e *Event, err error) {

	if msg == NoMsg {
		return e, getCommandError(MsgBusErrNoMsg)
	}

	if id == "" {
		return e, getCommandError(MsgBusErrNoID)
	}

	if data == nil {
		return e, getCommandError(MsgBusErrNoData)
	}

	c := cmd{
		op:      opPub,
		sync:    true,
		msg:     msg,
		ID:      id,
		data:    data,
		eventch: nil,
	}

	e, err = ps.dispatch(&c)

	return e, err
}

// Sub subscribes to a message/command, asynchronously.
func (ps *PubSub) Sub(msg MsgType, id IDString, ech EventChan) (requestID int, err error) {
	requestID = <-ps.requestIDChan

	if msg == NoMsg {
		return requestID, getCommandError(MsgBusErrNoMsg)
	}

	if ech == nil {
		return requestID, getCommandError(MsgBusErrNoEventChan)
	}

	c := cmd{
		op:        opSub,
		sync:      false,
		msg:       msg,
		ID:        id,
		requestID: requestID,
		data:      nil,
		eventch:   ech,
	}

	_, err = ps.dispatch(&c)

	return requestID, err
}

// SubWait subscribes to a message/command, synchronously.
func (ps *PubSub) SubWait(msg MsgType, id IDString, ech EventChan) (e *Event, err error) {

	if msg == NoMsg {
		return e, getCommandError(MsgBusErrNoMsg)
	}

	if ech == nil {
		return e, getCommandError(MsgBusErrNoEventChan)
	}

	c := cmd{
		op:      opSub,
		sync:    true,
		msg:     msg,
		ID:      id,
		data:    nil,
		eventch: ech,
	}

	return ps.dispatch(&c)

}

// Get retrieves a record by ID or, if no ID, then all
// record IDs associated with the given message type, asynchronously.
func (ps *PubSub) Get(msg MsgType, id IDString, ech EventChan) (requestID int, err error) {
	requestID = <-ps.requestIDChan

	if msg == NoMsg {
		return requestID, getCommandError(MsgBusErrNoMsg)
	}

	if ech == nil {
		return requestID, getCommandError(MsgBusErrNoEventChan)
	}

	c := cmd{
		op:        opGet,
		sync:      false,
		msg:       msg,
		ID:        id,
		requestID: requestID,
		data:      nil,
		eventch:   ech,
	}

	_, err = ps.dispatch(&c)

	return requestID, err

}

// GetWait retrieves a record by ID or, if no ID, then all
// record IDs associated with the given message type, synchronously.
func (ps *PubSub) GetWait(msg MsgType, id IDString) (e *Event, err error) {

	if msg == NoMsg {
		return e, getCommandError(MsgBusErrNoMsg)
	}

	c := cmd{
		op:      opGet,
		sync:    true,
		msg:     msg,
		ID:      id,
		data:    nil,
		eventch: nil,
	}

	return ps.dispatch(&c)

}

//--------------------------------------------------------------------------------
//
//--------------------------------------------------------------------------------
func (ps *PubSub) SearchIP(msg MsgType, ip string, ech EventChan) (requestID int, err error) {
	requestID = <-ps.requestIDChan

	if msg == NoMsg {
		return requestID, getCommandError(MsgBusErrNoMsg)
	}

	if ech == nil {
		return requestID, getCommandError(MsgBusErrNoEventChan)
	}

	if ip == "" {
		return requestID, getCommandError(MsgBusErrNoSearchTerm)
	}

	c := cmd{
		op:        opSearch,
		sync:      false,
		msg:       msg,
		IP:        ip,
		requestID: requestID,
		data:      nil,
		eventch:   ech,
	}

	_, err = ps.dispatch(&c)

	return requestID, err

}

//--------------------------------------------------------------------------------
//
//--------------------------------------------------------------------------------
func (ps *PubSub) SearchIPWait(msg MsgType, ip string) (e *Event, err error) {

	if msg == NoMsg {
		return e, getCommandError(MsgBusErrNoMsg)
	}

	if ip == "" {
		return e, getCommandError(MsgBusErrNoSearchTerm)
	}

	c := cmd{
		op:      opSearch,
		sync:    true,
		msg:     msg,
		IP:      ip,
		data:    nil,
		eventch: nil,
	}

	return ps.dispatch(&c)

}

//--------------------------------------------------------------------------------
//
//--------------------------------------------------------------------------------
func (ps *PubSub) SearchMAC(msg MsgType, mac string, ech EventChan) (requestID int, err error) {
	requestID = <-ps.requestIDChan

	if msg == NoMsg {
		return requestID, getCommandError(MsgBusErrNoMsg)
	}

	if ech == nil {
		return requestID, getCommandError(MsgBusErrNoEventChan)
	}

	if mac == "" {
		return requestID, getCommandError(MsgBusErrNoSearchTerm)
	}

	c := cmd{
		op:        opSearch,
		sync:      false,
		msg:       msg,
		MAC:       mac,
		data:      nil,
		eventch:   ech,
		requestID: requestID,
	}

	_, err = ps.dispatch(&c)

	return requestID, err

}

//--------------------------------------------------------------------------------
//
//--------------------------------------------------------------------------------
func (ps *PubSub) SearchMACWait(msg MsgType, mac string) (e *Event, err error) {

	if msg == NoMsg {
		return e, getCommandError(MsgBusErrNoMsg)
	}

	if mac == "" {
		return e, getCommandError(MsgBusErrNoSearchTerm)
	}

	c := cmd{
		op:      opSearch,
		sync:    true,
		msg:     msg,
		MAC:     mac,
		data:    nil,
		eventch: nil,
	}

	return ps.dispatch(&c)

}

//--------------------------------------------------------------------------------
//
//--------------------------------------------------------------------------------
func (ps *PubSub) SearchName(msg MsgType, name string, ech EventChan) (requestID int, err error) {
	requestID = <-ps.requestIDChan

	if msg == NoMsg {
		return requestID, getCommandError(MsgBusErrNoMsg)
	}

	if ech == nil {
		return requestID, getCommandError(MsgBusErrNoEventChan)
	}

	if name == "" {
		return requestID, getCommandError(MsgBusErrNoSearchTerm)
	}

	c := cmd{
		op:        opSearch,
		sync:      false,
		msg:       msg,
		Name:      name,
		data:      nil,
		eventch:   ech,
		requestID: requestID,
	}

	_, err = ps.dispatch(&c)

	return requestID, err

}

//--------------------------------------------------------------------------------
//
//--------------------------------------------------------------------------------
func (ps *PubSub) SearchNameWait(msg MsgType, name string) (e *Event, err error) {

	if msg == NoMsg {
		return e, getCommandError(MsgBusErrNoMsg)
	}

	if name == "" {
		return e, getCommandError(MsgBusErrNoSearchTerm)
	}

	c := cmd{
		op:      opSearch,
		sync:    true,
		msg:     msg,
		Name:    name,
		data:    nil,
		eventch: nil,
	}

	return ps.dispatch(&c)

}

//--------------------------------------------------------------------------------
//
//--------------------------------------------------------------------------------
func (ps *PubSub) Set(msg MsgType, id IDString, data interface{}) (requestID int, err error) {
	requestID = <-ps.requestIDChan

	if msg == NoMsg {
		return requestID, getCommandError(MsgBusErrNoMsg)
	}

	if id == "" {
		return requestID, getCommandError(MsgBusErrNoID)
	}

	if data == nil {
		return requestID, getCommandError(MsgBusErrNoData)
	}

	c := cmd{
		op:        opSet,
		sync:      false,
		msg:       msg,
		ID:        id,
		requestID: requestID,
		data:      data,
		eventch:   nil,
	}

	_, err = ps.dispatch(&c)

	return requestID, err
}

//--------------------------------------------------------------------------------
//
//--------------------------------------------------------------------------------
func (ps *PubSub) SetWait(msg MsgType, id IDString, data interface{}) (e *Event, err error) {

	if msg == NoMsg {
		return e, getCommandError(MsgBusErrNoMsg)
	}

	if id == "" {
		return e, getCommandError(MsgBusErrNoID)
	}

	if data == nil {
		return e, getCommandError(MsgBusErrNoData)
	}

	c := cmd{
		op:      opSet,
		sync:    true,
		msg:     msg,
		ID:      id,
		data:    data,
		eventch: nil,
	}

	return ps.dispatch(&c)

}

//--------------------------------------------------------------------------------
// Request removal of events for the topic
//--------------------------------------------------------------------------------
func (ps *PubSub) Unsub(msg MsgType, id IDString, ech EventChan) (requestID int, err error) {
	requestID = <-ps.requestIDChan

	if msg == NoMsg {
		return requestID, getCommandError(MsgBusErrNoMsg)
	}

	if id == "" {
		return requestID, getCommandError(MsgBusErrNoID)
	}

	if ech == nil {
		return requestID, getCommandError(MsgBusErrNoEventChan)
	}

	c := cmd{
		op:        opUnsub,
		sync:      false,
		msg:       msg,
		ID:        id,
		requestID: requestID,
		data:      nil,
		eventch:   ech,
	}

	_, err = ps.dispatch(&c)

	return requestID, err
}

//--------------------------------------------------------------------------------
// Request removal of events for the topic
//--------------------------------------------------------------------------------
func (ps *PubSub) UnsubWait(msg MsgType, id IDString, ech EventChan) (e *Event, err error) {

	if msg == NoMsg {
		return e, getCommandError(MsgBusErrNoMsg)
	}

	if id == "" {
		return e, getCommandError(MsgBusErrNoID)
	}

	if ech == nil {
		return e, getCommandError(MsgBusErrNoEventChan)
	}

	c := cmd{
		op:      opUnsub,
		sync:    true,
		msg:     msg,
		ID:      id,
		data:    nil,
		eventch: ech,
	}

	return ps.dispatch(&c)

}

//--------------------------------------------------------------------------------
// Request removal of the topic
//--------------------------------------------------------------------------------
func (ps *PubSub) Unpub(msg MsgType, id IDString, ech ...EventChan) (requestID int, err error) {
	requestID = <-ps.requestIDChan

	if msg == NoMsg {
		return requestID, getCommandError(MsgBusErrNoMsg)
	}

	if id == "" {
		return requestID, getCommandError(MsgBusErrNoID)
	}

	var eventchan EventChan = nil

	for _, v := range ech {
		if v != nil {
			eventchan = v
			break
		}
	}

	c := cmd{
		op:        opUnpub,
		sync:      false,
		msg:       msg,
		ID:        id,
		requestID: requestID,
		data:      nil,
		eventch:   eventchan,
	}

	_, err = ps.dispatch(&c)

	return requestID, err
}

//--------------------------------------------------------------------------------
// Request removal of the topic
//--------------------------------------------------------------------------------
func (ps *PubSub) UnpubWait(msg MsgType, id IDString) (e *Event, err error) {

	if msg == NoMsg {
		return e, getCommandError(MsgBusErrNoMsg)
	}

	if id == "" {
		return e, getCommandError(MsgBusErrNoID)
	}

	c := cmd{
		op:      opUnpub,
		sync:    true,
		msg:     msg,
		ID:      id,
		data:    nil,
		eventch: nil,
	}

	return ps.dispatch(&c)

}

//--------------------------------------------------------------------------------
// Request update events for the topic
//--------------------------------------------------------------------------------
func (ps *PubSub) RemoveAndCloseEventChan(ech EventChan) (requestID int, err error) {
	requestID = <-ps.requestIDChan

	if ech == nil {
		return requestID, getCommandError(MsgBusErrNoEventChan)
	}

	c := cmd{
		op:        opRemove,
		sync:      false,
		msg:       NoMsg,
		ID:        "",
		requestID: requestID,
		data:      nil,
		eventch:   ech,
	}

	_, err = ps.dispatch(&c)

	return requestID, err
}

//--------------------------------------------------------------------------------
// Request update events for the topic
//--------------------------------------------------------------------------------
func (ps *PubSub) RemoveAndCloseEventChanWait(ech EventChan) (e *Event, err error) {

	if ech == nil {
		return e, getCommandError(MsgBusErrNoEventChan)
	}

	c := cmd{
		op:      opRemove,
		sync:    true,
		msg:     NoMsg,
		ID:      "",
		data:    nil,
		eventch: ech,
	}

	return ps.dispatch(&c)

}

//--------------------------------------------------------------------------------
//
// Shutdown closes all subscribed channels and terminates the goroutine.
//
//--------------------------------------------------------------------------------
func (ps *PubSub) Shutdown() (requestID int, err error) {
	requestID = <-ps.requestIDChan

	c := cmd{
		op:        opShutdown,
		sync:      false,
		msg:       NoMsg,
		ID:        "",
		requestID: requestID,
		data:      nil,
		eventch:   nil,
	}

	_, err = ps.dispatch(&c)

	return requestID, err
}

//--------------------------------------------------------------------------------
//
// Shutdown closes all subscribed channels and terminates the goroutine.
//
//--------------------------------------------------------------------------------
func (ps *PubSub) ShutdownWait() (e *Event, err error) {

	c := cmd{
		op:      opShutdown,
		sync:    true,
		msg:     NoMsg,
		ID:      "",
		data:    nil,
		eventch: nil,
	}

	return ps.dispatch(&c)

}

//--------------------------------------------------------------------------------
//
//
//
//--------------------------------------------------------------------------------
func (ps *PubSub) dispatch(c *cmd) (event *Event, e error) {

	e = nil

	if c.op == opNop {
		return event, getCommandError(MsgBusErrNoErr)
	}

	if c.sync {
		c.returnch = make(EventChan)
	}

	ps.cmdChan <- c

	if c.sync {
		event = <-c.returnch
		e = event.Err
		close(c.returnch)
	}

	return event, e
}

//--------------------------------------------------------------------------------
//
//
//
//--------------------------------------------------------------------------------
func (ps *PubSub) start() {
	defer close(ps.requestIDChan)
	go func(requestIDChan chan<- int) {
		counter := 1
		for {
			select {
			case <-ps.done:
				break
			default:
				requestIDChan <- counter
				counter++
			}
		}
	}(ps.requestIDChan)

	reg := registry{
		data:   make(map[MsgType]map[IDString]registryData),
		notify: make(map[MsgType]map[chan *Event]interface{}),
	}

	reg.data[ConfigMsg] = make(map[IDString]registryData)
	reg.data[ContractManagerConfigMsg] = make(map[IDString]registryData)
	reg.data[DestMsg] = make(map[IDString]registryData)
	reg.data[NodeOperatorMsg] = make(map[IDString]registryData)
	reg.data[ContractMsg] = make(map[IDString]registryData)
	reg.data[MinerMsg] = make(map[IDString]registryData)
	reg.data[ConnectionMsg] = make(map[IDString]registryData)
	reg.data[ValidateMsg] = make(map[IDString]registryData)

	reg.notify[ConfigMsg] = make(map[chan *Event]interface{})
	reg.notify[ContractManagerConfigMsg] = make(map[chan *Event]interface{})
	reg.notify[DestMsg] = make(map[chan *Event]interface{})
	reg.notify[NodeOperatorMsg] = make(map[chan *Event]interface{})
	reg.notify[ContractMsg] = make(map[chan *Event]interface{})
	reg.notify[MinerMsg] = make(map[chan *Event]interface{})
	reg.notify[ConnectionMsg] = make(map[chan *Event]interface{})
	reg.notify[ValidateMsg] = make(map[chan *Event]interface{})

loop:
	for cmdptr := range ps.cmdChan {
		ps.logger.Logf(log.LevelInfo, "MSGBUS: %+v", *cmdptr)

		if cmdptr.op == opNop {
			continue loop
		}

		if cmdptr.op == opShutdown {
			break loop
		}

		switch cmdptr.op {
		case opPub:
			reg.pub(cmdptr)

		case opSub:
			reg.sub(cmdptr)

		case opSet:
			reg.set(cmdptr)

		case opGet:
			reg.get(cmdptr)

		case opSearch:
			reg.search(cmdptr)

		case opUnpub:
			reg.unpub(cmdptr)

		case opUnsub:
			reg.unsub(cmdptr)

		case opRemove:
			reg.removeAndClose(cmdptr)

		default:
			panic("default reached for cmd.op")
		}
	}

	ps.done <- struct{}{}

	fmt.Printf("Closing PubSub Command chan\n")

	// clean up here
	// Close any open channels
	// Delete registry

}

//-----------------------------------------
//
//-----------------------------------------
func (event *Event) send(e EventChan) {

	go func(e EventChan, event *Event) {
		e <- event
	}(e, event)

}

//-----------------------------------------
// msg contains the message type
// ID contains the new value
// data contains the new data struct
//
//-----------------------------------------
func (reg *registry) pub(c *cmd) {

	event := Event{
		EventType: PublishEvent,
		Msg:       c.msg,
		ID:        c.ID,
		RequestID: c.requestID,
		Data:      c.data,
		Err:       nil,
	}

	if _, ok := reg.data[c.msg]; !ok {
		reg.data[c.msg] = make(map[IDString]registryData)
	}

	if _, ok := reg.data[c.msg][c.ID]; ok {
		event.Err = getCommandError(MsgBusErrDupData)
	} else {
		reg.data[c.msg][c.ID] = registryData{
			sub:  Subscribers{eventchan: make(map[EventChan]int)},
			data: c.data,
		}
	}

	// If sync, return the event
	if c.sync {
		// sendEvent(c.returnch, event)
		event.send(c.returnch)
	}

	if c.eventch != nil {
		// sendEvent(c.eventch, event)
		event.send(c.eventch)
	}

	// If no error, copy the event to everyone interested
	if event.Err == nil {
		for ech := range reg.notify[c.msg] {
			//sendEvent(ech, event)
			event.send(ech)
		}
	}

}

//-----------------------------------------
// msg
// ID (optional)
// ch Event Channel
//
//-----------------------------------------
func (reg *registry) sub(c *cmd) {

	event := Event{
		EventType: SubscribedEvent,
		Msg:       c.msg,
		ID:        c.ID,
		RequestID: c.requestID,
		Data:      c.data,
		Err:       nil,
	}

	if c.ID == "" {
		if _, ok := reg.notify[c.msg]; !ok {
			event.Err = getCommandError(MsgBusErrBadMsg)
		} else if _, ok := reg.notify[c.msg][c.eventch]; ok {
			event.Err = getCommandError(MsgBusErrDupData)
		} else {
			reg.notify[c.msg][c.eventch] = 1
		}
	} else {
		if _, ok := reg.data[c.msg]; !ok {
			event.Err = getCommandError(MsgBusErrBadMsg)
		} else if _, ok := reg.data[c.msg][c.ID]; !ok {
			event.Err = getCommandError(MsgBusErrBadID)
		} else if _, ok := reg.data[c.msg][c.ID].sub.eventchan[c.eventch]; ok {
			event.Err = getCommandError(MsgBusErrDupData)
		} else {
			reg.data[c.msg][c.ID].sub.eventchan[c.eventch] = 1
		}
	}

	if c.sync {
		event.send(c.returnch)
	}

	if c.eventch != nil {
		event.send(c.eventch)
	}
}

//-----------------------------------------
//
//
//
//-----------------------------------------
func (reg *registry) set(c *cmd) {

	event := Event{
		EventType: UpdateEvent,
		Msg:       c.msg,
		ID:        c.ID,
		RequestID: c.requestID,
		Data:      c.data,
		Err:       nil,
	}

	if c.ID == "" {
		event.Err = getCommandError(MsgBusErrNoID)
		fmt.Printf(lumerinlib.FileLine()+"Error:%s\n", event.Err)
	} else if _, ok := reg.data[c.msg]; !ok {
		event.Err = getCommandError(MsgBusErrBadMsg)
		fmt.Printf(lumerinlib.FileLine()+"Error:%s\n", event.Err)
	} else if _, ok := reg.notify[c.msg]; !ok {
		event.Err = getCommandError(MsgBusErrBadMsg)
		fmt.Printf(lumerinlib.FileLine()+"Error:%s\n", event.Err)
	} else if _, ok := reg.data[c.msg][c.ID]; !ok {
		event.Err = getCommandError(MsgBusErrBadID)
		fmt.Printf(lumerinlib.FileLine()+"Error:%s\n", event.Err)
	} else {

		//
		// Need Error Checking on the in coming data type to match msgtype
		//

		// Could do a lot here to check if the data actually changed
		// Set the data

		d := reg.data[c.msg][c.ID]
		d.data = c.data
		reg.data[c.msg][c.ID] = d

	}

	if c.sync {
		event.send(c.returnch)
	}

	if c.eventch != nil {
		event.send(c.eventch)
	}

	// Notify anyone listening for the message class
	for nch := range reg.notify[c.msg] {
		event.send(nch)
	}
	// Notify anyone listening for the specific ID
	for ech := range reg.data[c.msg][c.ID].sub.eventchan {
		if _, ok := reg.data[c.msg][c.ID].sub.eventchan[ech]; ok {
			event.send(ech)
		} else {
			panic(fmt.Sprintf(lumerinlib.FileLine() + "Error eventchannel not ok"))
		}
	}
}

//-----------------------------------------
//
// Msg
// ID
// eventch - where to send the get request to
//
//-----------------------------------------
func (reg *registry) get(c *cmd) {

	event := Event{
		EventType: GetEvent,
		Msg:       c.msg,
		ID:        c.ID,
		RequestID: c.requestID,
		Data:      nil,
		Err:       nil,
	}

	if _, ok := reg.data[c.msg]; !ok {
		event.Err = getCommandError(MsgBusErrBadMsg)
	} else if c.ID == "" {
		var index IDIndex
		for i := range reg.data[c.msg] {
			index = append(index, i)
		}
		event.EventType = GetIndexEvent
		event.Data = index
	} else if _, ok := reg.data[c.msg][c.ID]; !ok {
		event.Err = getCommandError(MsgBusErrBadID)
	} else {
		event.Data = reg.data[c.msg][c.ID].data
	}

	if c.sync {
		event.send(c.returnch)
	}
	if c.eventch != nil {
		event.send(c.eventch)
	}

}

//-----------------------------------------
//
//
//-----------------------------------------
func (reg *registry) search(c *cmd) {

	event := Event{
		EventType: SearchEvent,
		Msg:       c.msg,
		Data:      nil,
		Err:       nil,
		RequestID: c.requestID,
	}

	// Only works for Miner messages at the moment.
	if c.msg != MinerMsg {
		event.Err = getCommandError(MsgBusErrBadMsg)
		return
	}

	if c.Name == "" && c.IP == "" && c.MAC != "" {
		event.Err = getCommandError(MsgBusErrBadSearchTerm)
		return
	}

	if _, ok := reg.data[c.msg]; !ok {
		event.Err = getCommandError(MsgBusErrNoData)
		return
	}

	var index IDIndex
	switch {
	case c.Name != "":
		for i := range reg.data[c.msg] {
			if reg.data[c.msg][i].data.(Miner).Name == c.Name {
				index = append(index, i)
			}
		}
	case c.IP != "":
		for i := range reg.data[c.msg] {
			if reg.data[c.msg][i].data.(Miner).IP == c.IP {
				index = append(index, i)
			}
		}
	case c.MAC != "":
		for i := range reg.data[c.msg] {
			if reg.data[c.msg][i].data.(Miner).MAC == c.MAC {
				index = append(index, i)
			}
		}
	default:
		panic("")
	}
	event.EventType = SearchIndexEvent
	event.Data = index

	if c.sync {
		event.send(c.returnch)
	}
	if c.eventch != nil {
		event.send(c.eventch)
	}

}

//-----------------------------------------
//
//
//
//-----------------------------------------
func (reg *registry) unsub(c *cmd) {

	event := Event{
		EventType: UnsubscribedEvent,
		Msg:       c.msg,
		ID:        c.ID,
		RequestID: c.requestID,
		Data:      nil,
		Err:       nil,
	}

	if _, ok := reg.data[c.msg]; !ok {
		event.Err = getCommandError(MsgBusErrBadMsg)
	} else if _, ok := reg.notify[c.msg]; !ok {
		event.Err = getCommandError(MsgBusErrBadMsg)
	} else if c.ID == "" {
		if _, ok := reg.notify[c.msg][c.eventch]; !ok {
			event.Err = getCommandError(MsgBusErrNoSub)
		} else {
			delete(reg.notify[c.msg], c.eventch)
		}
	} else {
		if _, ok := reg.data[c.msg][c.ID]; !ok {
			event.Err = getCommandError(MsgBusErrBadID)
		} else if _, ok := reg.data[c.msg][c.ID].sub.eventchan[c.eventch]; !ok {
			event.Err = getCommandError(MsgBusErrNoSub)
		} else {
			delete(reg.data[c.msg][c.ID].sub.eventchan, c.eventch)
		}
	}

	if c.sync {
		event.send(c.returnch)
	}
	if c.eventch != nil {
		event.send(c.eventch)
	}

}

//-----------------------------------------
//
//-----------------------------------------
func (reg *registry) unpub(c *cmd) {

	event := Event{
		EventType: UnpublishEvent,
		Msg:       c.msg,
		ID:        c.ID,
		RequestID: c.requestID,
		Data:      nil,
		Err:       nil,
	}

	if _, ok := reg.data[c.msg]; !ok {
		event.Err = getCommandError(MsgBusErrBadMsg)
	} else if _, ok := reg.notify[c.msg]; !ok {
		event.Err = getCommandError(MsgBusErrBadMsg)
	} else if c.ID == "" {
		event.Err = getCommandError(MsgBusErrNoID)
	} else if _, ok := reg.data[c.msg][c.ID]; !ok {
		event.Err = getCommandError(MsgBusErrBadID)
	}

	if c.sync {
		event.send(c.returnch)
	}

	if c.eventch != nil {
		event.send(c.eventch)
	}

	for ech := range reg.data[c.msg][c.ID].sub.eventchan {
		event.send(ech)
	}

	for ech := range reg.notify[c.msg] {
		event.send(ech)
	}

	delete(reg.data[c.msg], c.ID)

}

//---------------------------------------
//
//	data   map[MsgType]map[IDString]registryData
//	notify map[MsgType]map[chan Event]interface{}
//
//---------------------------------------
func (reg *registry) removeAndClose(c *cmd) {

	event := Event{
		EventType: RemovedEvent,
		Msg:       c.msg,
		ID:        c.ID,
		RequestID: c.requestID,
		Data:      nil,
		Err:       nil,
	}

	if c.eventch == nil {
		event.Err = getCommandError(MsgBusErrNoEventChan)
	}

	for msg := range reg.notify {
		delete(reg.notify[msg], c.eventch)
	}

	for msg := range reg.data {
		for id := range reg.data[msg] {
			if _, ok := reg.data[msg][id].sub.eventchan[c.eventch]; ok {
				delete(reg.data[c.msg][c.ID].sub.eventchan, c.eventch)
			}
		}
	}

	close(c.eventch)

	if c.sync {
		event.send(c.returnch)
	}

}
