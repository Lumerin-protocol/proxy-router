package validator

import (
	"testing"
)

//test to see if a channel is successfully made
func TestChannelCreation(t *testing.T) {
	c := Channels{
		ValidationChannels: make(map[string]chan Message),
	}

	c.AddChannel("hello")

	_, res := c.GetChannel("hello")

	if !res {
		t.Error("failed to receive hello channel")
	}
}
