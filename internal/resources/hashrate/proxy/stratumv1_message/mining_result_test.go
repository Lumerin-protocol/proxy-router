package stratumv1_message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJobNotFound(t *testing.T) {
	ID := 1
	msg := NewMiningResultJobNotFound(ID)

	msg, err := ParseMiningResult(msg.Serialize())

	assert.NoError(t, err)
	assert.Equal(t, msg.ID, ID)
	assert.True(t, msg.IsError())
	assert.Equal(t, msg.GetError(), `["21","Job not found"]`)
}

func TestMessageWithResultFalse(t *testing.T) {
	ID := 47
	msg := NewMiningResultFalse(ID)

	msg, err := ParseMiningResult(msg.Serialize())
	assert.NoError(t, err)
	assert.Equal(t, msg.ID, ID)
	assert.True(t, msg.IsError())
}
