package contracts

import (
	"github.com/Lumerin-protocol/contracts-go/v3/aggregatorv3interface"
	"github.com/Lumerin-protocol/contracts-go/v3/futures"
	"github.com/Lumerin-protocol/contracts-go/v3/hashrateoracle"
	"github.com/Lumerin-protocol/contracts-go/v3/ierc20"
	"github.com/Lumerin-protocol/contracts-go/v3/multicall3"
	"github.com/Lumerin-protocol/proxy-router/internal/lib"
)

// AllContractsMeta is an array of all contracts metadata to be used for decoding errors
var AllContractsMeta = []lib.Meta{
	futures.FuturesMetaData,
	multicall3.Multicall3MetaData,
	hashrateoracle.HashrateoracleMetaData,
	ierc20.Ierc20MetaData,
	aggregatorv3interface.Aggregatorv3interfaceMetaData,
}
