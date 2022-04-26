package msgdata

import (
	"errors"
	"fmt"

	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

// Struct of Contract parameters in JSON
type ContractJSON struct {
	IsSeller               bool   `json:"isSeller"`
	ID                     string `json:"id"`
	State                  string `json:"state"`
	Buyer                  string `json:"buyer"`
	Price                  int    `json:"price"`
	Limit                  int    `json:"limit"`
	Speed                  int    `json:"speed"`
	Length                 int    `json:"length"`
	StartingBlockTimestamp int    `json:"startingBlockTimestamp"`
	Dest                   string `json:"dest"`
}

//Struct that stores slice of all JSON Contract structs in Repo
type ContractRepo struct {
	ContractJSONs []ContractJSON
	Ps            *msgbus.PubSub
}

//Initialize Repo with empty slice of JSON Contract structs
func NewContract(ps *msgbus.PubSub) *ContractRepo {
	return &ContractRepo{
		ContractJSONs: []ContractJSON{},
		Ps:            ps,
	}
}

//Return all Contract Structs in Repo
func (r *ContractRepo) GetAllContracts() []ContractJSON {
	return r.ContractJSONs
}

//Return Contract Struct by ID
func (r *ContractRepo) GetContract(id string) (ContractJSON, error) {
	for i, c := range r.ContractJSONs {
		if c.ID == id {
			return r.ContractJSONs[i], nil
		}
	}
	return r.ContractJSONs[0], errors.New("ID not found")
}

//Add a new Contract Struct to to Repo
func (r *ContractRepo) AddContract(contract ContractJSON) {
	r.ContractJSONs = append(r.ContractJSONs, contract)
}

//Converts Contract struct from msgbus to JSON struct and adds it to Repo
func (r *ContractRepo) AddContractFromMsgBus(contractID msgbus.ContractID, contract msgbus.Contract) {
	var contractJSON ContractJSON

	contractJSON.IsSeller = contract.IsSeller
	contractJSON.ID = string(contractID)
	contractJSON.State = string(contract.State)
	contractJSON.Buyer = string(contract.Buyer)
	contractJSON.Price = contract.Price
	contractJSON.Limit = contract.Limit
	contractJSON.Speed = contract.Speed
	contractJSON.Length = contract.Length
	contractJSON.StartingBlockTimestamp = contract.StartingBlockTimestamp
	contractJSON.Dest = string(contract.Dest)

	r.ContractJSONs = append(r.ContractJSONs, contractJSON)
}

//Update Contract Struct with specific ID and leave empty parameters unchanged
func (r *ContractRepo) UpdateContract(id string, newContract ContractJSON) error {
	for i, c := range r.ContractJSONs {
		if c.ID == id {
			r.ContractJSONs[i].IsSeller = newContract.IsSeller
			if newContract.State != "" {
				r.ContractJSONs[i].State = newContract.State
			}
			if newContract.Buyer != "" {
				r.ContractJSONs[i].Buyer = newContract.Buyer
			}
			if newContract.Price != 0 {
				r.ContractJSONs[i].Price = newContract.Price
			}
			if newContract.Limit != 0 {
				r.ContractJSONs[i].Limit = newContract.Limit
			}
			if newContract.Speed != 0 {
				r.ContractJSONs[i].Speed = newContract.Speed
			}
			if newContract.Length != 0 {
				r.ContractJSONs[i].Length = newContract.Length
			}
			if newContract.StartingBlockTimestamp != 0 {
				r.ContractJSONs[i].StartingBlockTimestamp = newContract.StartingBlockTimestamp
			}
			if newContract.Dest != "" {
				r.ContractJSONs[i].Dest = newContract.Dest
			}

			return nil
		}
	}
	return errors.New("ID not found")
}

//Delete Contract Struct with specific ID
func (r *ContractRepo) DeleteContract(id string) error {
	for i, c := range r.ContractJSONs {
		if c.ID == id {
			r.ContractJSONs = append(r.ContractJSONs[:i], r.ContractJSONs[i+1:]...)

			return nil
		}
	}
	return errors.New("ID not found")
}

//Subscribe to events for contract msgs on msgbus to update API repos with data
func (r *ContractRepo) SubscribeToContractMsgBus() {
	contractCh := msgbus.NewEventChan()

	// add existing contracts to api repo
	event, err := r.Ps.GetWait(msgbus.ContractMsg, "")
	if err != nil {
		panic(fmt.Sprintf("Getting Contracts Failed: %s", err))
	}
	contracts := event.Data.(msgbus.IDIndex)
	if len(contracts) > 0 {
		for i := range contracts {
			event, err = r.Ps.GetWait(msgbus.ContractMsg, msgbus.IDString(contracts[i]))
			if err != nil {
				panic(fmt.Sprintf("Getting Contract Failed: %s", err))
			}
			contract := event.Data.(msgbus.Contract)
			r.AddContractFromMsgBus(msgbus.ContractID(contracts[i]), contract)
		}
	}

	event, err = r.Ps.SubWait(msgbus.ContractMsg, "", contractCh)
	if err != nil {
		panic(fmt.Sprintf("SubWait failed: %s\n", err))
	}
	if event.EventType != msgbus.SubscribedEvent {
		panic(fmt.Sprintf("Wrong event type %v\n", event))
	}

	for event = range contractCh {
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
			contractID := msgbus.ContractID(event.ID)

			// do not push to api repo if it already exists
			for i := range r.ContractJSONs {
				if r.ContractJSONs[i].ID == string(contractID) {
					break loop
				}
			}
			contract := event.Data.(msgbus.Contract)
			r.AddContractFromMsgBus(contractID, contract)

			//
			// Delete/Unpublish Event
			//
		case msgbus.DeleteEvent:
			fallthrough
		case msgbus.UnpublishEvent:
			fmt.Printf(lumerinlib.Funcname()+" Delete/Unpublish Event: %v\n", event)
			contractID := msgbus.ContractID(event.ID)
			r.DeleteContract(string(contractID))

			//
			// Update Event
			//
		case msgbus.UpdateEvent:
			fmt.Printf(lumerinlib.Funcname()+" Update Event: %v\n", event)
			contractID := msgbus.ContractID(event.ID)
			contract := event.Data.(msgbus.Contract)
			contractJSON := ConvertContractMSGtoContractJSON(contract)
			r.UpdateContract(string(contractID), contractJSON)

			//
			// Rut Row...
			//
		default:
			fmt.Printf(lumerinlib.Funcname()+" Got Event: %v\n", event)
		}
	}
}

func ConvertContractJSONtoContractMSG(contract ContractJSON) msgbus.Contract {
	var msg msgbus.Contract

	msg.IsSeller = contract.IsSeller
	msg.ID = msgbus.ContractID(contract.ID)
	msg.State = msgbus.ContractState(contract.State)
	msg.Buyer = contract.Buyer
	msg.Price = contract.Price
	msg.Limit = contract.Limit
	msg.Speed = contract.Speed
	msg.Length = contract.Length
	msg.StartingBlockTimestamp = contract.StartingBlockTimestamp
	msg.Dest = msgbus.DestID(contract.Dest)

	return msg
}

func ConvertContractMSGtoContractJSON(msg msgbus.Contract) (contract ContractJSON) {
	contract.IsSeller = msg.IsSeller
	contract.ID = string(msg.ID)
	contract.State = string(msg.State)
	contract.Buyer = string(msg.Buyer)
	contract.Price = msg.Price
	contract.Limit = msg.Limit
	contract.Speed = msg.Speed
	contract.Length = msg.Length
	contract.StartingBlockTimestamp = msg.StartingBlockTimestamp
	contract.Dest = string(msg.Dest)

	return contract
}
