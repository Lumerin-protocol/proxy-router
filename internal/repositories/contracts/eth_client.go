package contracts

import (
	"context"
	"net/url"

	"github.com/ethereum/go-ethereum/ethclient"
)

type EthClient struct {
	// config
	url string

	// state
	*ethclient.Client
	supportsSubscriptions bool
}

func DialContext(ctx context.Context, urlString string) (*EthClient, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	isHTTP := u.Scheme == "http" || u.Scheme == "https"

	client, err := ethclient.DialContext(ctx, urlString)
	if err != nil {
		return nil, err
	}
	return &EthClient{
		Client:                client,
		url:                   urlString,
		supportsSubscriptions: isHTTP,
	}, nil
}

func (c *EthClient) SupportsSubscriptions() bool {
	return c.supportsSubscriptions
}
