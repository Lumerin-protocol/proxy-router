package connections

import (
	"encoding/json"

	"gitlab.com/TitanInd/lumerin/interfaces"
)

type Connection struct {
	interfaces.IConnectionModel `json:"-"`
	Id                          string
	isAvailable                 bool
	Address                     string `json:"ipAddress"`
	Destination                 string `json:"socketAddress"`
}

func (m *Connection) GetId() string {
	return m.Id
}

func (m *Connection) AddressesMatch(address string, destination string) bool {
	return m.Address == address && m.Destination == destination
}

func (c *Connection) GetAvailable() bool {
	return c.isAvailable
}

func (c *Connection) SetAvailable(isAvailable bool) {
	c.isAvailable = isAvailable
}

func (c *Connection) MarshalJSON() ([]byte, error) {
	type Alias Connection
	status := "Running"

	if c.GetAvailable() {
		status = "Available"
	}

	return json.Marshal(&struct {
		Status string `json:"status"`
		*Alias
	}{
		Status: status,
		Alias:  (*Alias)(c),
	})
}
