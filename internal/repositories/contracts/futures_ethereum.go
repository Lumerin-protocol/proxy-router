package contracts

import (
	"context"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/Lumerin-protocol/contracts-go/v3/futures"
	"github.com/Lumerin-protocol/proxy-router/internal/interfaces"
	"github.com/Lumerin-protocol/proxy-router/internal/lib"
	"github.com/Lumerin-protocol/proxy-router/internal/repositories/multicall"
	mc "github.com/Lumerin-protocol/proxy-router/internal/repositories/multicall"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type FuturesEthereum struct {
	// config
	legacyTx    bool // use legacy transaction fee, for local node testing
	futuresAddr common.Address

	// state
	nonce      uint64
	mutex      lib.Mutex
	futuresABI *abi.ABI

	// deps
	futures    *futures.Futures
	multicall  *multicall.Multicall3Custom
	client     EthereumClient
	logWatcher LogWatcher
	log        interfaces.ILogger
}

func NewFuturesEthereum(futuresAddr common.Address, multicalladdr common.Address, client EthereumClient, logWatcher LogWatcher, log interfaces.ILogger) *FuturesEthereum {
	ft, err := futures.NewFutures(futuresAddr, client)
	if err != nil {
		panic("invalid clonefactory ABI")
	}
	ftABI, err := futures.FuturesMetaData.GetAbi()
	if err != nil {
		panic("invalid futures ABI: " + err.Error())
	}

	multicall := mc.NewMulticall3Custom(client, multicalladdr)

	return &FuturesEthereum{
		futures:     ft,
		multicall:   multicall,
		futuresAddr: futuresAddr,
		client:      client,
		futuresABI:  ftABI,
		mutex:       lib.NewMutex(),
		logWatcher:  logWatcher,
		log:         log,
	}
}

func (g *FuturesEthereum) SetLegacyTx(legacyTx bool) {
	g.legacyTx = legacyTx
}

func (g *FuturesEthereum) GetToken(ctx context.Context) (common.Address, error) {
	return g.futures.Token(&bind.CallOpts{Context: ctx})
}

// GetOngoingDeliveryRange returns the start and end of the ongoing delivery range
func (g *FuturesEthereum) GetOngoingDeliveryRange(ctx context.Context) (start time.Time, end time.Time, duration time.Duration, err error) {
	_firstDeliveryDate, err := g.futures.FirstFutureDeliveryDate(&bind.CallOpts{Context: ctx})
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}

	_deliveryIntervalDays, err := g.futures.DeliveryIntervalDays(&bind.CallOpts{Context: ctx})
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}

	duration = time.Duration(_deliveryIntervalDays) * 24 * time.Hour
	firstDeliveryDate := time.Unix(_firstDeliveryDate.Int64(), 0)

	start, end = GetOngoingDeliveryRange(firstDeliveryDate, duration, time.Now())
	return start, end, duration, nil
}

func (g *FuturesEthereum) GetMatchedContracts(ctx context.Context, userAddress common.Address, deliveryDate time.Time) ([]FuturesContract, error) {
	posIDs, err := g.futures.GetPositionsByParticipantDeliveryDate(&bind.CallOpts{Context: ctx}, userAddress, big.NewInt(deliveryDate.Unix()))
	if err != nil {
		return nil, err
	}

	args := make([][]any, len(posIDs))
	for i, id := range posIDs {
		args[i] = []any{id}
	}

	pos, err := mc.Batch[futures.FuturesPosition](ctx, g.multicall, g.futuresABI, g.futuresAddr, "getPositionById", args)
	if err != nil {
		return nil, err
	}

	contracts := make([]FuturesContract, len(pos))
	for i, position := range pos {
		contracts[i] = FuturesContract{
			ContractID: posIDs[i],
			Seller:     position.Seller,
			Buyer:      position.Buyer,
			DestURL:    position.DestURL,
			DeliveryAt: time.Unix(position.DeliveryAt.Int64(), 0),
			Paid:       position.Paid,
		}
	}

	return contracts, nil
}

func (g *FuturesEthereum) CloseDelivery(ctx context.Context, positionID common.Hash, blameSeller bool, privKey string) error {
	timeout := 2 * time.Minute
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	l := lib.NewMutex()

	err := l.LockCtx(ctx)
	if err != nil {
		err = lib.WrapError(err, fmt.Errorf("close contract lock error %s", timeout))
		g.log.Error(err)
		return err
	}
	defer l.Unlock()

	err = g.closeDelivery(ctx, positionID, privKey, blameSeller)
	if err != nil {
		if strings.Contains(err.Error(), "the contract is not in the running state") {
			return ErrNotRunning
		}
		return lib.WrapError(fmt.Errorf("close contract error"), err)
	}

	return err
}

type CloseDeliveryReq struct {
	PositionID  common.Hash
	BlameSeller bool
}

func (g *FuturesEthereum) BatchCloseDelivery(ctx context.Context, reqs []CloseDeliveryReq, privKey string) error {
	transactOpts, err := g.getTransactOpts(ctx, privKey)
	if err != nil {
		g.log.Error(err)
		return err
	}

	calls := make([][]byte, len(reqs))
	for i, req := range reqs {
		calls[i], err = g.futuresABI.Pack("closeDelivery", req.PositionID, req.BlameSeller)
		if err != nil {
			g.log.Error(err)
			return err
		}
	}

	tx, err := g.futures.Multicall(transactOpts, calls)
	if err != nil {
		g.log.Error(err)
		return err
	}

	_, err = bind.WaitMined(ctx, g.client, tx)
	if err != nil {
		g.log.Error(err)
		return err
	}

	return nil
}

func (g *FuturesEthereum) GetPosition(ctx context.Context, positionID common.Hash) (*FuturesContract, error) {
	pos, err := g.futures.GetPositionById(&bind.CallOpts{Context: ctx}, positionID)
	if err != nil {
		return nil, err
	}

	return &FuturesContract{
		ContractID: positionID,
		Seller:     pos.Seller,
		Buyer:      pos.Buyer,
		DestURL:    pos.DestURL,
		DeliveryAt: time.Unix(pos.DeliveryAt.Int64(), 0),
		Paid:       pos.Paid,
	}, nil
}

func (g *FuturesEthereum) closeDelivery(ctx context.Context, positionID common.Hash, privKey string, blameSeller bool) error {
	transactOpts, err := g.getTransactOpts(ctx, privKey)
	if err != nil {
		g.log.Error(err)
		return err
	}

	tx, err := g.futures.CloseDelivery(transactOpts, (positionID), blameSeller)
	if err != nil {
		return err
	}

	g.log.Debugf("closed delivery, position id %s, blame seller %b nonce %d", positionID, blameSeller, tx.Nonce())

	_, err = bind.WaitMined(ctx, g.client, tx)
	if err != nil {
		g.log.Error(err)
		return err
	}

	return nil
}

type ContractSpecs struct {
	ValidatorAddress common.Address
	ValidatorURL     *url.URL
	DeliveryDuration time.Duration
	SpeedHps         uint64
}

func (g *FuturesEthereum) GetContractSpecs(ctx context.Context) (*ContractSpecs, error) {
	validatorAddress, err := g.futures.ValidatorAddress(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}
	if validatorAddress == (common.Address{}) {
		return nil, fmt.Errorf("validator address is not set on the futures contract")
	}

	validatorURLString, err := g.futures.ValidatorURL(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}

	validatorURL, err := url.Parse(validatorURLString)
	if err != nil {
		return nil, err
	}

	deliveryDurationDays, err := g.futures.DeliveryDurationDays(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}

	speedHps, err := g.futures.SpeedHps(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}

	return &ContractSpecs{
		ValidatorAddress: validatorAddress,
		ValidatorURL:     validatorURL,
		DeliveryDuration: time.Duration(deliveryDurationDays) * time.Hour * 24,
		SpeedHps:         speedHps.Uint64(),
	}, nil
}

func (g *FuturesEthereum) CreateFuturesSubscription(ctx context.Context, futuresAddr common.Address) (*lib.Subscription, error) {
	return g.logWatcher.Watch(ctx, futuresAddr, CreateEventMapper(futuresEventFactory, g.futuresABI), nil)
}

func (g *FuturesEthereum) getTransactOpts(ctx context.Context, privKey string) (*bind.TransactOpts, error) {
	privateKey, err := crypto.HexToECDSA(privKey)
	if err != nil {
		return nil, err
	}

	chainId, err := g.client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	transactOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		return nil, err
	}

	// TODO: deal with likely gasPrice issue so our transaction processes before another pending nonce.
	if g.legacyTx {
		gasPrice, err := g.client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, err
		}
		transactOpts.GasPrice = gasPrice
	}

	transactOpts.Value = big.NewInt(0)
	transactOpts.Context = ctx

	return transactOpts, nil
}

func GetOngoingDeliveryRange(firstDeliveryDate time.Time, deliveryInterval time.Duration, now time.Time) (start time.Time, end time.Time) {
	duration := deliveryInterval
	elapsed := now.Sub(firstDeliveryDate)
	index := elapsed / duration
	start = firstDeliveryDate.Add(index * duration)
	end = start.Add(duration)
	return start, end
}
