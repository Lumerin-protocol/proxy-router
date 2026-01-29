package stratumv1_message

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMiningExtranonceSubscribe(t *testing.T) {
	msg := NewMiningExtranonceSubscribe(1)
	b := msg.Serialize()
	t.Logf("msg: %s", string(b))
	require.NotNil(t, b)
	require.Equal(t, MethodExtranonceSubscribe, msg.Method)
	require.Nil(t, msg.Params)
}

func TestMiningExtranonceSubscribeParse(t *testing.T) {
	msg := NewMiningExtranonceSubscribe(1)
	b := msg.Serialize()
	parsed, err := ParseMiningExtranonceSubscribe(b)
	require.NoError(t, err)
	require.Equal(t, msg.Method, parsed.Method)
	require.Equal(t, msg.Params, parsed.Params)
	require.Equal(t, msg.ID, parsed.ID)
}
