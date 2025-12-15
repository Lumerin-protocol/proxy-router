package subgraph

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGetAllPositions(t *testing.T) {
	client := NewClient("https://graphidx.dev.lumerin.io/subgraphs/name/marketplace")
	deliveryAt := time.Unix(1763679600, 0)
	positions, err := client.GetAllPositions(context.Background(), deliveryAt)
	fmt.Printf("%+v\n", positions)
	require.NoError(t, err)
	require.NotNil(t, positions)
}

func TestGetPositionsBySeller(t *testing.T) {
	client := NewClient("https://graphidx.dev.lumerin.io/subgraphs/name/marketplace")
	participantAddress := common.HexToAddress("0xb4b12a69fdbb70b31214d4d3c063752c186ff8de")
	deliveryAt := time.Unix(1763679600, 0)
	positions, err := client.GetPositionsBySeller(context.Background(), participantAddress, deliveryAt)
	fmt.Printf("%+v\n", positions)
	require.NoError(t, err)
	require.NotNil(t, positions)
}
