package msgdata

import (
	"errors"
	"fmt"

	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

//Struct of NodeOperator parameters in JSON
type NodeOperatorJSON struct {
	ID                     string                                     `json:"id"`
	DefaultDest            string                                     `json:"defaultDest"`
	EthereumAccount        string                                     `json:"ethereumAccount"`
	TotalAvailableHashRate int                                        `json:"totalAvailableHashrate"`
	UnusedHashRate         int                                        `json:"unusedHashRate"`
	Contracts              map[msgbus.ContractID]msgbus.ContractState `json:"contracts"`
}

//Struct that stores slice of all JSON NodeOperator structs in Repo
type NodeOperatorRepo struct {
	NodeOperatorJSONs []NodeOperatorJSON
	Ps                *msgbus.PubSub
}

//Initialize Repo with empty slice of JSON NodeOperator structs
func NewNodeOperator(ps *msgbus.PubSub) *NodeOperatorRepo {
	return &NodeOperatorRepo{
		NodeOperatorJSONs: []NodeOperatorJSON{},
		Ps:                ps,
	}
}

//Return all NodeOperator Structs in Repo
func (r *NodeOperatorRepo) GetAllNodeOperators() []NodeOperatorJSON {
	return r.NodeOperatorJSONs
}

//Return NodeOperator Struct by ID
func (r *NodeOperatorRepo) GetNodeOperator(id string) (NodeOperatorJSON, error) {
	for i, d := range r.NodeOperatorJSONs {
		if d.ID == id {
			return r.NodeOperatorJSONs[i], nil
		}
	}
	return r.NodeOperatorJSONs[0], errors.New("ID not found")
}

//Add a new NodeOperator Struct to to Repo
func (r *NodeOperatorRepo) AddNodeOperator(nodeOperator NodeOperatorJSON) {
	r.NodeOperatorJSONs = append(r.NodeOperatorJSONs, nodeOperator)
}

//Converts NodeOperator struct from msgbus to JSON struct and adds it to Repo
func (r *NodeOperatorRepo) AddNodeOperatorFromMsgBus(nodeOperatorID msgbus.NodeOperatorID, nodeOperator msgbus.NodeOperator) {
	var nodeOperatorJSON NodeOperatorJSON

	nodeOperatorJSON.ID = string(nodeOperatorID)
	nodeOperatorJSON.DefaultDest = string(nodeOperator.DefaultDest)
	nodeOperatorJSON.EthereumAccount = nodeOperator.EthereumAccount
	nodeOperatorJSON.TotalAvailableHashRate = nodeOperator.TotalAvailableHashRate
	nodeOperatorJSON.UnusedHashRate = nodeOperator.UnusedHashRate
	nodeOperatorJSON.Contracts = nodeOperator.Contracts

	r.NodeOperatorJSONs = append(r.NodeOperatorJSONs, nodeOperatorJSON)
}

//Update NodeOperator Struct with specific ID and leave empty parameters unchanged
func (r *NodeOperatorRepo) UpdateNodeOperator(id string, newNodeOperator NodeOperatorJSON) error {
	for i, d := range r.NodeOperatorJSONs {
		if d.ID == id {
			if newNodeOperator.DefaultDest != "" {
				r.NodeOperatorJSONs[i].DefaultDest = newNodeOperator.DefaultDest
			}
			if newNodeOperator.EthereumAccount != "" {
				r.NodeOperatorJSONs[i].EthereumAccount = newNodeOperator.EthereumAccount
			}
			if newNodeOperator.TotalAvailableHashRate != 0 {
				r.NodeOperatorJSONs[i].TotalAvailableHashRate = newNodeOperator.TotalAvailableHashRate
			}
			if newNodeOperator.UnusedHashRate != 0 {
				r.NodeOperatorJSONs[i].UnusedHashRate = newNodeOperator.UnusedHashRate
			}
			if newNodeOperator.Contracts != nil {
				r.NodeOperatorJSONs[i].Contracts = newNodeOperator.Contracts
			}
			return nil
		}
	}
	return errors.New("ID not found")
}

//Delete NodeOperator Struct with specific ID
func (r *NodeOperatorRepo) DeleteNodeOperator(id string) error {
	for i, d := range r.NodeOperatorJSONs {
		if d.ID == id {
			r.NodeOperatorJSONs = append(r.NodeOperatorJSONs[:i], r.NodeOperatorJSONs[i+1:]...)

			return nil
		}
	}
	return errors.New("ID not found")
}

//Subscribe to events for nodeOperator msgs on msgbus to update API repos with data
func (r *NodeOperatorRepo) SubscribeToNodeOperatorMsgBus() {
	nodeOperatorCh := msgbus.NewEventChan()

	// add existing nodeOperators to api repo
	event, err := r.Ps.GetWait(msgbus.NodeOperatorMsg, "")
	if err != nil {
		panic(fmt.Sprintf("Getting NodeOperators Failed: %s", err))
	}
	nodeOperators := event.Data.(msgbus.IDIndex)
	if len(nodeOperators) > 0 {
		for i := range nodeOperators {
			event, err = r.Ps.GetWait(msgbus.NodeOperatorMsg, msgbus.IDString(nodeOperators[i]))
			if err != nil {
				panic(fmt.Sprintf("Getting NodeOperator Failed: %s", err))
			}
			nodeOperator := event.Data.(msgbus.NodeOperator)
			r.AddNodeOperatorFromMsgBus(msgbus.NodeOperatorID(nodeOperators[i]), nodeOperator)
		}
	}

	event, err = r.Ps.SubWait(msgbus.NodeOperatorMsg, "", nodeOperatorCh)
	if err != nil {
		panic(fmt.Sprintf("SubWait failed: %s\n", err))
	}
	if event.EventType != msgbus.SubscribedEvent {
		panic(fmt.Sprintf("Wrong event type %v\n", event))
	}

	for event = range nodeOperatorCh {
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
			nodeOperatorID := msgbus.NodeOperatorID(event.ID)

			// do not push to api repo if it already exists
			for i := range r.NodeOperatorJSONs {
				if r.NodeOperatorJSONs[i].ID == string(nodeOperatorID) {
					break loop
				}
			}
			nodeOperator := event.Data.(msgbus.NodeOperator)
			r.AddNodeOperatorFromMsgBus(nodeOperatorID, nodeOperator)

			//
			// Delete/Unpublish Event
			//
		case msgbus.DeleteEvent:
			fallthrough
		case msgbus.UnpublishEvent:
			fmt.Printf(lumerinlib.Funcname()+" Delete/Unpublish Event: %v\n", event)
			nodeOperatorID := msgbus.NodeOperatorID(event.ID)
			r.DeleteNodeOperator(string(nodeOperatorID))

			//
			// Update Event
			//
		case msgbus.UpdateEvent:
			fmt.Printf(lumerinlib.Funcname()+" Update Event: %v\n", event)
			nodeOperatorID := msgbus.NodeOperatorID(event.ID)
			nodeOperator := event.Data.(msgbus.NodeOperator)
			nodeOperatorJSON := ConvertNodeOperatorMSGtoNodeOperatorJSON(nodeOperator)
			r.UpdateNodeOperator(string(nodeOperatorID), nodeOperatorJSON)

			//
			// Rut Row...
			//
		default:
			fmt.Printf(lumerinlib.Funcname()+" Got Event: %v\n", event)
		}
	}
}

func ConvertNodeOperatorJSONtoNodeOperatorMSG(nodeOperator NodeOperatorJSON) msgbus.NodeOperator {
	var msg msgbus.NodeOperator

	msg.ID = msgbus.NodeOperatorID(nodeOperator.ID)
	msg.DefaultDest = msgbus.DestID(nodeOperator.DefaultDest)
	msg.EthereumAccount = nodeOperator.EthereumAccount
	msg.TotalAvailableHashRate = nodeOperator.TotalAvailableHashRate
	msg.UnusedHashRate = nodeOperator.UnusedHashRate
	msg.Contracts = nodeOperator.Contracts

	return msg
}

func ConvertNodeOperatorMSGtoNodeOperatorJSON(msg msgbus.NodeOperator) (nodeOperator NodeOperatorJSON) {
	nodeOperator.ID = string(msg.ID)
	nodeOperator.DefaultDest = string(msg.DefaultDest)
	nodeOperator.EthereumAccount = msg.EthereumAccount
	nodeOperator.TotalAvailableHashRate = msg.TotalAvailableHashRate
	nodeOperator.UnusedHashRate = msg.UnusedHashRate
	nodeOperator.Contracts = msg.Contracts

	return nodeOperator
}
