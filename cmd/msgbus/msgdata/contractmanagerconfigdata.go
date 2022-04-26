package msgdata

import (
	"errors"
	"fmt"

	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

// Struct of ContractManagerConfig parameters in JSON
type ContractManagerConfigJSON struct {
	ID                  string `json:"id"`
	Mnemonic            string `json:"mnemonic"`
	AccountIndex        int    `json:"accountIndex"`
	EthNodeAddr         string `json:"ethNodeAddr"`
	ClaimFunds          bool   `json:"claimFunds"`
	TimeThreshold       int    `json:"timeThreshold"`
	CloneFactoryAddress string `json:"cloneFactoryAddress"`
	LumerinTokenAddress string `json:"lumerinTokenAddress"`
	ValidatorAddress    string `json:"validatorAddress"`
	ProxyAddress        string `json:"proxyAddress"`
}

//Struct that stores slice of all JSON ContractManagerConfig structs in Repo
type ContractManagerConfigRepo struct {
	ContractManagerConfigJSONs []ContractManagerConfigJSON
	Ps                         *msgbus.PubSub
}

//Initialize Repo with empty slice of JSON ContractManagerConfig structs
func NewContractManagerConfig(ps *msgbus.PubSub) *ContractManagerConfigRepo {
	return &ContractManagerConfigRepo{
		ContractManagerConfigJSONs: []ContractManagerConfigJSON{},
		Ps:                         ps,
	}
}

//Return all ContractManagerConfig Structs in Repo
func (r *ContractManagerConfigRepo) GetAllContractManagerConfigs() []ContractManagerConfigJSON {
	return r.ContractManagerConfigJSONs
}

//Return ContractManagerConfig Struct by ID
func (r *ContractManagerConfigRepo) GetContractManagerConfig(id string) (ContractManagerConfigJSON, error) {
	for i, c := range r.ContractManagerConfigJSONs {
		if c.ID == id {
			return r.ContractManagerConfigJSONs[i], nil
		}
	}
	return r.ContractManagerConfigJSONs[0], errors.New("ID not found")
}

//Add a new ContractManagerConfig Struct to to Repo
func (r *ContractManagerConfigRepo) AddContractManagerConfig(contractConf ContractManagerConfigJSON) {
	r.ContractManagerConfigJSONs = append(r.ContractManagerConfigJSONs, contractConf)
}

//Converts ContractManagerConfig struct from msgbus to JSON struct and adds it to Repo
func (r *ContractManagerConfigRepo) AddContractManagerConfigFromMsgBus(contractConfID msgbus.ContractManagerConfigID, contractConf msgbus.ContractManagerConfig) {
	var contractConfJSON ContractManagerConfigJSON

	contractConfJSON.ID = string(contractConfID)
	contractConfJSON.Mnemonic = string(contractConf.Mnemonic)
	contractConfJSON.AccountIndex = int(contractConf.AccountIndex)
	contractConfJSON.EthNodeAddr = string(contractConf.EthNodeAddr)
	contractConfJSON.ClaimFunds = bool(contractConf.ClaimFunds)
	contractConfJSON.TimeThreshold = int(contractConf.TimeThreshold)
	contractConfJSON.CloneFactoryAddress = string(contractConf.CloneFactoryAddress)
	contractConfJSON.LumerinTokenAddress = string(contractConf.LumerinTokenAddress)
	contractConfJSON.ValidatorAddress = string(contractConf.ValidatorAddress)
	contractConfJSON.ProxyAddress = string(contractConf.ProxyAddress)

	r.ContractManagerConfigJSONs = append(r.ContractManagerConfigJSONs, contractConfJSON)
}

//Update ContractManagerConfig Struct with specific ID and leave empty parameters unchanged
func (r *ContractManagerConfigRepo) UpdateContractManagerConfig(id string, newContractManagerConfig ContractManagerConfigJSON) error {
	for i, c := range r.ContractManagerConfigJSONs {
		if c.ID == id {
			if newContractManagerConfig.Mnemonic != "" {
				r.ContractManagerConfigJSONs[i].Mnemonic = newContractManagerConfig.Mnemonic
			}
			r.ContractManagerConfigJSONs[i].AccountIndex = newContractManagerConfig.AccountIndex
			if newContractManagerConfig.EthNodeAddr != "" {
				r.ContractManagerConfigJSONs[i].EthNodeAddr = newContractManagerConfig.EthNodeAddr
			}
			r.ContractManagerConfigJSONs[i].ClaimFunds = newContractManagerConfig.ClaimFunds
			r.ContractManagerConfigJSONs[i].TimeThreshold = newContractManagerConfig.TimeThreshold
			if newContractManagerConfig.CloneFactoryAddress != "" {
				r.ContractManagerConfigJSONs[i].CloneFactoryAddress = newContractManagerConfig.CloneFactoryAddress
			}
			if newContractManagerConfig.LumerinTokenAddress != "" {
				r.ContractManagerConfigJSONs[i].LumerinTokenAddress = newContractManagerConfig.LumerinTokenAddress
			}
			if newContractManagerConfig.ValidatorAddress != "" {
				r.ContractManagerConfigJSONs[i].ValidatorAddress = newContractManagerConfig.ValidatorAddress
			}
			if newContractManagerConfig.ProxyAddress != "" {
				r.ContractManagerConfigJSONs[i].ProxyAddress = newContractManagerConfig.ProxyAddress
			}

			return nil
		}
	}
	return errors.New("ID not found")
}

//Delete ContractManagerConfig Struct with specific ID
func (r *ContractManagerConfigRepo) DeleteContractManagerConfig(id string) error {
	for i, c := range r.ContractManagerConfigJSONs {
		if c.ID == id {
			r.ContractManagerConfigJSONs = append(r.ContractManagerConfigJSONs[:i], r.ContractManagerConfigJSONs[i+1:]...)

			return nil
		}
	}
	return errors.New("ID not found")
}

//Subscribe to events for contractConfig msgs on msgbus to update API repos with data
func (r *ContractManagerConfigRepo) SubscribeToContractManagerConfigMsgBus() {
	contractConfigCh := msgbus.NewEventChan()

	// add existing contractConfigs to api repo
	event, err := r.Ps.GetWait(msgbus.ContractManagerConfigMsg, "")
	if err != nil {
		panic(fmt.Sprintf("Getting Contract Manager Configs Failed: %s", err))
	}
	contractConfigs := event.Data.(msgbus.IDIndex)
	if len(contractConfigs) > 0 {
		for i := range contractConfigs {
			event, err = r.Ps.GetWait(msgbus.ContractManagerConfigMsg, msgbus.IDString(contractConfigs[i]))
			if err != nil {
				panic(fmt.Sprintf("Getting Contract Manager Config Failed: %s", err))
			}
			contractConfig := event.Data.(msgbus.ContractManagerConfig)
			r.AddContractManagerConfigFromMsgBus(msgbus.ContractManagerConfigID(contractConfigs[i]), contractConfig)
		}
	}

	event, err = r.Ps.SubWait(msgbus.ContractManagerConfigMsg, "", contractConfigCh)
	if err != nil {
		panic(fmt.Sprintf("SubWait failed: %s\n", err))
	}
	if event.EventType != msgbus.SubscribedEvent {
		panic(fmt.Sprintf("Wrong event type %v\n", event))
	}

	for event = range contractConfigCh {
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
			contractConfigID := msgbus.ContractManagerConfigID(event.ID)

			// do not push to api repo if it already exists
			for i := range r.ContractManagerConfigJSONs {
				if r.ContractManagerConfigJSONs[i].ID == string(contractConfigID) {
					break loop
				}
			}
			contractConfig := event.Data.(msgbus.ContractManagerConfig)
			r.AddContractManagerConfigFromMsgBus(contractConfigID, contractConfig)

			//
			// Delete/Unpublish Event
			//
		case msgbus.DeleteEvent:
			fallthrough
		case msgbus.UnpublishEvent:
			fmt.Printf(lumerinlib.Funcname()+" Delete/Unpublish Event: %v\n", event)
			contractConfigID := msgbus.ContractManagerConfigID(event.ID)
			r.DeleteContractManagerConfig(string(contractConfigID))

			//
			// Update Event
			//
		case msgbus.UpdateEvent:
			fmt.Printf(lumerinlib.Funcname()+" Update Event: %v\n", event)
			contractConfigID := msgbus.ContractManagerConfigID(event.ID)
			contractConfig := event.Data.(msgbus.ContractManagerConfig)
			contractConfigJSON := ConvertContractManagerConfigMSGtoContractManagerConfigJSON(contractConfig)
			r.UpdateContractManagerConfig(string(contractConfigID), contractConfigJSON)

			//
			// Rut Row...
			//
		default:
			fmt.Printf(lumerinlib.Funcname()+" Got Event: %v\n", event)
		}
	}
}

func ConvertContractManagerConfigJSONtoContractManagerConfigMSG(contractConf ContractManagerConfigJSON) msgbus.ContractManagerConfig {
	var msg msgbus.ContractManagerConfig

	msg.ID = msgbus.ContractManagerConfigID(contractConf.ID)
	msg.Mnemonic = contractConf.Mnemonic
	msg.AccountIndex = contractConf.AccountIndex
	msg.EthNodeAddr = contractConf.EthNodeAddr
	msg.ClaimFunds = contractConf.ClaimFunds
	msg.TimeThreshold = contractConf.TimeThreshold
	msg.CloneFactoryAddress = contractConf.CloneFactoryAddress
	msg.LumerinTokenAddress = contractConf.LumerinTokenAddress
	msg.ValidatorAddress = contractConf.ValidatorAddress
	msg.ProxyAddress = contractConf.ProxyAddress

	return msg
}

func ConvertContractManagerConfigMSGtoContractManagerConfigJSON(msg msgbus.ContractManagerConfig) (contractConf ContractManagerConfigJSON) {
	contractConf.ID = string(msg.ID)
	contractConf.Mnemonic = msg.Mnemonic
	contractConf.AccountIndex = msg.AccountIndex
	contractConf.EthNodeAddr = msg.EthNodeAddr
	contractConf.ClaimFunds = msg.ClaimFunds
	contractConf.TimeThreshold = msg.TimeThreshold
	contractConf.CloneFactoryAddress = msg.CloneFactoryAddress
	contractConf.LumerinTokenAddress = msg.LumerinTokenAddress
	contractConf.ValidatorAddress = msg.ValidatorAddress
	contractConf.ProxyAddress = msg.ProxyAddress

	return contractConf
}
