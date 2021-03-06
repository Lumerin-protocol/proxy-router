package channels

import (
	"example.com/message"
)

//contains a mapping of channels to communicate with goroutines
type Channels struct {
	ValidationChannels map[string]chan message.Message
}

//function to add a channel to the ValidationChannels variable
func (c *Channels) AddChannel(ethAddress string) chan message.Message {
	c.ValidationChannels[ethAddress] = make(chan message.Message)
	return c.ValidationChannels[ethAddress]
}

//receives a channel based on the ethereum address
func (c *Channels) GetChannel(ethAddress string) (chan message.Message, bool) {
	channel, ok := c.ValidationChannels[ethAddress]
	return channel, ok
}
