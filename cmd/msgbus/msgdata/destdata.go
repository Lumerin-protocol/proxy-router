package msgdata

import (
	"errors"
	"fmt"

	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

// Struct of Dest parameters in JSON
type DestJSON struct {
	ID     string `json:"id"`
	NetUrl string `json:"netUrl"`
}

//Struct that stores slice of all JSON Dest structs in Repo
type DestRepo struct {
	DestJSONs []DestJSON
	Ps        *msgbus.PubSub
}

//Initialize Repo with empty slice of JSON Dest structs
func NewDest(ps *msgbus.PubSub) *DestRepo {
	return &DestRepo{
		DestJSONs: []DestJSON{},
		Ps:        ps,
	}
}

//Return all Dest Structs in Repo
func (r *DestRepo) GetAllDests() []DestJSON {
	return r.DestJSONs
}

//Return Dest Struct by ID
func (r *DestRepo) GetDest(id string) (DestJSON, error) {
	for i, d := range r.DestJSONs {
		if d.ID == id {
			return r.DestJSONs[i], nil
		}
	}
	return r.DestJSONs[0], errors.New("ID not found")
}

//Add a new Dest Struct to to Repo
func (r *DestRepo) AddDest(dest DestJSON) {
	r.DestJSONs = append(r.DestJSONs, dest)
}

//Converts Dest struct from msgbus to JSON struct and adds it to Repo
func (r *DestRepo) AddDestFromMsgBus(destID msgbus.DestID, dest msgbus.Dest) {
	var destJSON DestJSON

	destJSON.ID = string(destID)
	destJSON.NetUrl = string(dest.NetUrl)

	r.DestJSONs = append(r.DestJSONs, destJSON)
}

//Update Dest Struct with specific ID and leave empty parameters unchanged
func (r *DestRepo) UpdateDest(id string, newDest DestJSON) error {
	for i, d := range r.DestJSONs {
		if d.ID == id {
			if newDest.NetUrl != "" {
				r.DestJSONs[i].NetUrl = newDest.NetUrl
			}

			return nil
		}
	}
	return errors.New("ID not found")
}

//Delete Dest Struct with specific ID
func (r *DestRepo) DeleteDest(id string) error {
	for i, d := range r.DestJSONs {
		if d.ID == id {
			r.DestJSONs = append(r.DestJSONs[:i], r.DestJSONs[i+1:]...)

			return nil
		}
	}
	return errors.New("ID not found")
}

//Subscribe to events for dest msgs on msgbus to update API repos with data
func (r *DestRepo) SubscribeToDestMsgBus() {
	destCh := msgbus.NewEventChan()

	// add existing dests to api repo
	event, err := r.Ps.GetWait(msgbus.DestMsg, "")
	if err != nil {
		panic(fmt.Sprintf("Getting Dests Failed: %s", err))
	}
	dests := event.Data.(msgbus.IDIndex)
	if len(dests) > 0 {
		for i := range dests {
			event, err = r.Ps.GetWait(msgbus.DestMsg, msgbus.IDString(dests[i]))
			if err != nil {
				panic(fmt.Sprintf("Getting Dest Failed: %s", err))
			}
			dest := event.Data.(msgbus.Dest)
			r.AddDestFromMsgBus(msgbus.DestID(dests[i]), dest)
		}
	}

	event, err = r.Ps.SubWait(msgbus.DestMsg, "", destCh)
	if err != nil {
		panic(fmt.Sprintf("SubWait failed: %s\n", err))
	}
	if event.EventType != msgbus.SubscribedEvent {
		panic(fmt.Sprintf("Wrong event type %v\n", event))
	}

	for event = range destCh {
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
			destID := msgbus.DestID(event.ID)

			// do not push to api repo if it already exists
			for i := range r.DestJSONs {
				if r.DestJSONs[i].ID == string(destID) {
					break loop
				}
			}
			dest := event.Data.(msgbus.Dest)
			r.AddDestFromMsgBus(destID, dest)

			//
			// Delete/Unpublish Event
			//
		case msgbus.DeleteEvent:
			fallthrough
		case msgbus.UnpublishEvent:
			fmt.Printf(lumerinlib.Funcname()+" Delete/Unpublish Event: %v\n", event)
			destID := msgbus.DestID(event.ID)
			r.DeleteDest(string(destID))

			//
			// Update Event
			//
		case msgbus.UpdateEvent:
			fmt.Printf(lumerinlib.Funcname()+" Update Event: %v\n", event)
			destID := msgbus.DestID(event.ID)
			dest := event.Data.(msgbus.Dest)
			destJSON := ConvertDestMSGtoDestJSON(dest)
			r.UpdateDest(string(destID), destJSON)

			//
			// Rut Row...
			//
		default:
			fmt.Printf(lumerinlib.Funcname()+" Got Event: %v\n", event)
		}
	}

}

func ConvertDestJSONtoDestMSG(dest DestJSON) msgbus.Dest {
	var msg msgbus.Dest

	msg.ID = msgbus.DestID(dest.ID)
	msg.NetUrl = msgbus.DestNetUrl(dest.NetUrl)

	return msg
}

func ConvertDestMSGtoDestJSON(msg msgbus.Dest) (dest DestJSON) {
	dest.ID = string(msg.ID)
	dest.NetUrl = string(msg.NetUrl)

	return dest
}
