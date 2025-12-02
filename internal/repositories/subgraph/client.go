package subgraph

import (
	"context"
	"strconv"
	"time"

	"github.com/Lumerin-protocol/proxy-router/internal/repositories/contracts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shurcooL/graphql"
)

type SubgraphClient struct {
	client *graphql.Client
}

func NewClient(subgraphURL string) *SubgraphClient {
	client := graphql.NewClient(subgraphURL, nil)
	return &SubgraphClient{
		client: client,
	}
}

func (c *SubgraphClient) GetAllPositions(ctx context.Context, deliveryAt time.Time) ([]contracts.FuturesContract, error) {
	var query struct {
		Positions []Position `graphql:"positions(where: {deliveryAt: $deliveryAt})"`
	}

	var variables = map[string]any{
		"deliveryAt": graphql.Int(deliveryAt.Unix()),
	}

	err := c.client.Query(ctx, &query, variables)
	if err != nil {
		return nil, err
	}

	return mapPositionsToContracts(query.Positions), nil
}

func (c *SubgraphClient) GetPositionsBySeller(ctx context.Context, sellerAddr common.Address, deliveryAt time.Time) ([]contracts.FuturesContract, error) {
	var query struct {
		Positions []Position `graphql:"positions(where: {seller_: {address: $sellerAddr}, deliveryAt: $deliveryAt})"`
	}

	var variables = map[string]any{
		"deliveryAt": graphql.Int(deliveryAt.Unix()),
		"sellerAddr": graphql.String(sellerAddr.Hex()),
	}

	err := c.client.Query(ctx, &query, variables)
	if err != nil {
		return nil, err
	}

	return mapPositionsToContracts(query.Positions), nil
}

func mapPositionsToContracts(positions []Position) []contracts.FuturesContract {
	contracts := make([]contracts.FuturesContract, len(positions))
	for i, position := range positions {
		contracts[i] = mapPositionToContract(position)
	}
	return contracts
}

func mapPositionToContract(position Position) contracts.FuturesContract {
	return contracts.FuturesContract{
		ContractID: common.HexToHash(position.ID),
		Seller:     common.HexToAddress(position.Seller.Address),
		Buyer:      common.HexToAddress(position.Buyer.Address),
		DestURL:    position.DestURL,
		DeliveryAt: time.Unix(mustAtoi(position.DeliveryAt), 0),
		Paid:       position.IsPaid,
	}
}

func mustAtoi(s string) int64 {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return int64(i)
}
