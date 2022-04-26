package msgdata

import (
	"errors"
	"fmt"
	"time"

	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

// Struct of Connection parameters in JSON
type ConnectionJSON struct {
	ID        string    `json:"id"`
	Miner     string    `json:"miner"`
	Dest      string    `json:"dest"`
	State     string    `json:"state"`
	TotalHash int       `json:"totalHash"`
	StartDate time.Time `json:"startDate"`
}

//Struct that stores slice of all JSON Connection structs in Repo
type ConnectionRepo struct {
	ConnectionJSONs []ConnectionJSON
	EventChan       msgbus.EventChan
	Ps              *msgbus.PubSub
}

type WebsocketMsg struct {
	Type            string           `json:"type"`
	ConnectionJSONs []ConnectionJSON `json:"connections"`
}

//Initialize Repo with empty slice of JSON Connection structs
func NewConnection(ps *msgbus.PubSub) *ConnectionRepo {
	return &ConnectionRepo{
		ConnectionJSONs: []ConnectionJSON{},
		Ps:              ps,
	}
}

//Return all Connection Structs in Repo
func (r *ConnectionRepo) GetAllConnections() []ConnectionJSON {
	return r.ConnectionJSONs
}

//Return Connection Struct by ID
func (r *ConnectionRepo) GetConnection(id string) (ConnectionJSON, error) {
	for i, c := range r.ConnectionJSONs {
		if c.ID == id {
			return r.ConnectionJSONs[i], nil
		}
	}
	return r.ConnectionJSONs[0], errors.New("ID not found")
}

//Add a new Connection Struct to to Repo
func (r *ConnectionRepo) AddConnection(conn ConnectionJSON) {
	r.ConnectionJSONs = append(r.ConnectionJSONs, conn)
}

//Converts Connection struct from msgbus to JSON struct and adds it to Repo
func (r *ConnectionRepo) AddConnectionFromMsgBus(connID msgbus.ConnectionID, conn msgbus.Connection) {
	var connJSON ConnectionJSON

	connJSON.ID = string(connID)
	connJSON.Miner = string(conn.Miner)
	connJSON.Dest = string(conn.Dest)
	connJSON.State = string(conn.State)
	connJSON.TotalHash = conn.TotalHash
	connJSON.StartDate = conn.StartDate

	r.ConnectionJSONs = append(r.ConnectionJSONs, connJSON)
}

//Update Connection Struct with specific ID and leave empty parameters unchanged
func (r *ConnectionRepo) UpdateConnection(id string, newConnection ConnectionJSON) error {
	for i, c := range r.ConnectionJSONs {
		if c.ID == id {
			if newConnection.Miner != "" {
				r.ConnectionJSONs[i].Miner = newConnection.Miner
			}
			if newConnection.Dest != "" {
				r.ConnectionJSONs[i].Dest = newConnection.Dest
			}
			if newConnection.State != "" {
				r.ConnectionJSONs[i].State = newConnection.State
			}
			if newConnection.TotalHash != 0 {
				r.ConnectionJSONs[i].TotalHash = newConnection.TotalHash
			}
			r.ConnectionJSONs[i].StartDate = newConnection.StartDate

			return nil
		}
	}
	return errors.New("ID not found")
}

//Delete Connection Struct with specific ID
func (r *ConnectionRepo) DeleteConnection(id string) error {
	for i, c := range r.ConnectionJSONs {
		if c.ID == id {
			r.ConnectionJSONs = append(r.ConnectionJSONs[:i], r.ConnectionJSONs[i+1:]...)

			return nil
		}
	}
	return errors.New("ID not found")
}

//Subscribe to events for connection msgs on msgbus to update API repos with data
func (r *ConnectionRepo) SubscribeToConnectionMsgBus() {
	r.EventChan = msgbus.NewEventChan()

	// add existing connections to api repo
	event, err := r.Ps.GetWait(msgbus.ConnectionMsg, "")
	if err != nil {
		panic(fmt.Sprintf("Getting Connections Failed: %s", err))
	}
	connections := event.Data.(msgbus.IDIndex)
	if len(connections) > 0 {
		for i := range connections {
			event, err = r.Ps.GetWait(msgbus.ConnectionMsg, msgbus.IDString(connections[i]))
			if err != nil {
				panic(fmt.Sprintf("Getting Connection Failed: %s", err))
			}
			connection := event.Data.(msgbus.Connection)
			r.AddConnectionFromMsgBus(msgbus.ConnectionID(connections[i]), connection)
		}
	}

	event, err = r.Ps.SubWait(msgbus.ConnectionMsg, "", r.EventChan)
	if err != nil {
		panic(fmt.Sprintf("SubWait failed: %s\n", err))
	}
	if event.EventType != msgbus.SubscribedEvent {
		panic(fmt.Sprintf("Wrong event type %v\n", event))
	}

	for event = range r.EventChan {
	loop:
		switch event.EventType {
		//
		// Subscribe Event
		//
		case msgbus.SubscribedEvent:
			fmt.Printf(lumerinlib.Funcname()+" Subscribe Event: %v\n", event)

			//
			// Publish Event
			//
		case msgbus.PublishEvent:
			fmt.Printf(lumerinlib.Funcname()+" Publish Event: %v\n", event)
			connectionID := msgbus.ConnectionID(event.ID)

			// do not push to api repo if it already exists
			for i := range r.ConnectionJSONs {
				if r.ConnectionJSONs[i].ID == string(connectionID) {
					break loop
				}
			}
			connection := event.Data.(msgbus.Connection)
			r.AddConnectionFromMsgBus(connectionID, connection)

			//
			// Delete/Unpublish Event
			//
		case msgbus.DeleteEvent:
			fallthrough
		case msgbus.UnpublishEvent:
			fmt.Printf(lumerinlib.Funcname()+" Delete/Unpublish Event: %v\n", event)
			connectionID := msgbus.ConnectionID(event.ID)
			r.DeleteConnection(string(connectionID))

			//
			// Update Event
			//
		case msgbus.UpdateEvent:
			fmt.Printf(lumerinlib.Funcname()+" Update Event: %v\n", event)
			connectionID := msgbus.ConnectionID(event.ID)
			connection := event.Data.(msgbus.Connection)
			connectionJSON := ConvertConnectionMSGtoConnectionJSON(connection)
			r.UpdateConnection(string(connectionID), connectionJSON)

			//
			// Rut Row...
			//
		default:
			fmt.Printf(lumerinlib.Funcname()+" Got Event: %v\n", event)
		}
	}
}

func ConvertConnectionJSONtoConnectionMSG(conn ConnectionJSON) msgbus.Connection {
	var msg msgbus.Connection

	msg.ID = msgbus.ConnectionID(conn.ID)
	msg.Miner = msgbus.MinerID(conn.Miner)
	msg.Dest = msgbus.DestID(conn.Dest)
	msg.State = msgbus.ConnectionState(conn.State)
	msg.TotalHash = conn.TotalHash
	msg.StartDate = conn.StartDate

	return msg
}

func ConvertConnectionMSGtoConnectionJSON(msg msgbus.Connection) (connection ConnectionJSON) {
	connection.ID = string(msg.ID)
	connection.Miner = string(msg.Miner)
	connection.Dest = string(msg.Dest)
	connection.State = string(msg.State)
	connection.TotalHash = msg.TotalHash
	connection.StartDate = msg.StartDate

	return connection
}
