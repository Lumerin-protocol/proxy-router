package msgdata

import (
	"errors"
	"fmt"

	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

// Struct of ConfigInfo parameters in JSON
type ConfigInfoJSON struct {
	ID           string `json:"id"`
	DefaultDest  string `json:"defaultDest"`
	NodeOperator string `json:"nodeOperator"`
}

//Struct that stores slice of all JSON ConfigInfo structs in Repo
type ConfigInfoRepo struct {
	ConfigInfoJSONs []ConfigInfoJSON
	Ps              *msgbus.PubSub
}

//Initialize Repo with empty slice of JSON ConfigInfo structs
func NewConfigInfo(ps *msgbus.PubSub) *ConfigInfoRepo {
	return &ConfigInfoRepo{
		ConfigInfoJSONs: []ConfigInfoJSON{},
		Ps:              ps,
	}
}

//Return all ConfigInfo Structs in Repo
func (r *ConfigInfoRepo) GetAllConfigInfos() []ConfigInfoJSON {
	return r.ConfigInfoJSONs
}

//Return ConfigInfo Struct by ID
func (r *ConfigInfoRepo) GetConfigInfo(id string) (ConfigInfoJSON, error) {
	for i, c := range r.ConfigInfoJSONs {
		if c.ID == id {
			return r.ConfigInfoJSONs[i], nil
		}
	}
	return r.ConfigInfoJSONs[0], errors.New("ID not found")
}

//Add a new ConfigInfo Struct to to Repo
func (r *ConfigInfoRepo) AddConfigInfo(conf ConfigInfoJSON) {
	r.ConfigInfoJSONs = append(r.ConfigInfoJSONs, conf)
}

//Converts ConfigInfo struct from msgbus to JSON struct and adds it to Repo
func (r *ConfigInfoRepo) AddConfigInfoFromMsgBus(confID msgbus.ConfigID, conf msgbus.ConfigInfo) {
	var confJSON ConfigInfoJSON

	confJSON.ID = string(confID)
	confJSON.DefaultDest = string(conf.DefaultDest)
	confJSON.NodeOperator = string(conf.NodeOperator)

	r.ConfigInfoJSONs = append(r.ConfigInfoJSONs, confJSON)
}

//Update ConfigInfo Struct with specific ID and leave empty parameters unchanged
func (r *ConfigInfoRepo) UpdateConfigInfo(id string, newConfigInfo ConfigInfoJSON) error {
	for i, c := range r.ConfigInfoJSONs {
		if c.ID == id {
			if newConfigInfo.DefaultDest != "" {
				r.ConfigInfoJSONs[i].DefaultDest = newConfigInfo.DefaultDest
			}
			if newConfigInfo.NodeOperator != "" {
				r.ConfigInfoJSONs[i].NodeOperator = newConfigInfo.NodeOperator
			}

			return nil
		}
	}
	return errors.New("ID not found")
}

//Delete ConfigInfo Struct with specific ID
func (r *ConfigInfoRepo) DeleteConfigInfo(id string) error {
	for i, c := range r.ConfigInfoJSONs {
		if c.ID == id {
			r.ConfigInfoJSONs = append(r.ConfigInfoJSONs[:i], r.ConfigInfoJSONs[i+1:]...)

			return nil
		}
	}
	return errors.New("ID not found")
}

//Subscribe to events for config msgs on msgbus to update API repos with data
func (r *ConfigInfoRepo) SubscribeToConfigInfoMsgBus() {
	configCh := msgbus.NewEventChan()

	// add existing configs to api repo
	event, err := r.Ps.GetWait(msgbus.ConfigMsg, "")
	if err != nil {
		panic(fmt.Sprintf("Getting Configs Failed: %s", err))
	}
	configs := event.Data.(msgbus.IDIndex)
	if len(configs) > 0 {
		for i := range configs {
			event, err = r.Ps.GetWait(msgbus.ConfigMsg, msgbus.IDString(configs[i]))
			if err != nil {
				panic(fmt.Sprintf("Getting Config Failed: %s", err))
			}
			config := event.Data.(msgbus.ConfigInfo)
			r.AddConfigInfoFromMsgBus(msgbus.ConfigID(configs[i]), config)
		}
	}

	event, err = r.Ps.SubWait(msgbus.ConfigMsg, "", configCh)
	if err != nil {
		panic(fmt.Sprintf("SubWait failed: %s\n", err))
	}
	if event.EventType != msgbus.SubscribedEvent {
		panic(fmt.Sprintf("Wrong event type %v\n", event))
	}

	for event = range configCh {
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
			configID := msgbus.ConfigID(event.ID)

			// do not push to api repo if it already exists
			for i := range r.ConfigInfoJSONs {
				if r.ConfigInfoJSONs[i].ID == string(configID) {
					break loop
				}
			}
			config := event.Data.(msgbus.ConfigInfo)
			r.AddConfigInfoFromMsgBus(configID, config)

			//
			// Delete/Unpublish Event
			//
		case msgbus.DeleteEvent:
			fallthrough
		case msgbus.UnpublishEvent:
			fmt.Printf(lumerinlib.Funcname()+" Delete/Unpublish Event: %v\n", event)
			configID := msgbus.ConfigID(event.ID)
			r.DeleteConfigInfo(string(configID))

			//
			// Update Event
			//
		case msgbus.UpdateEvent:
			fmt.Printf(lumerinlib.Funcname()+" Update Event: %v\n", event)
			configID := msgbus.ConfigID(event.ID)
			config := event.Data.(msgbus.ConfigInfo)
			configJSON := ConvertConfigInfoMSGtoConfigInfoJSON(config)
			r.UpdateConfigInfo(string(configID), configJSON)

			//
			// Rut Row...
			//
		default:
			fmt.Printf(lumerinlib.Funcname()+" Got Event: %v\n", event)
		}
	}
}

func ConvertConfigInfoJSONtoConfigInfoMSG(conf ConfigInfoJSON) msgbus.ConfigInfo {
	var msg msgbus.ConfigInfo

	msg.ID = msgbus.ConfigID(conf.ID)
	msg.DefaultDest = msgbus.DestID(conf.DefaultDest)
	msg.NodeOperator = msgbus.NodeOperatorID(conf.NodeOperator)

	return msg
}

func ConvertConfigInfoMSGtoConfigInfoJSON(msg msgbus.ConfigInfo) (conf ConfigInfoJSON) {
	conf.ID = string(msg.ID)
	conf.DefaultDest = string(msg.DefaultDest)
	conf.NodeOperator = string(msg.NodeOperator)

	return conf
}
