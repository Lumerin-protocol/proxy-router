package peervalidator

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"time"

	vr "github.com/Lumerin-protocol/contracts-go/validatorregistry"
	"github.com/Lumerin-protocol/proxy-router/internal/interfaces"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/rand"
)

type PeerValidator struct {
	interval     time.Duration // interval between checking peers aliveness
	registryAddr common.Address
	walletAddr   common.Address
	registry     *vr.Validatorregistry
	log          interfaces.ILogger
}

func NewPeerValidator(registryAddress common.Address, walletAddr common.Address, backend bind.ContractBackend, interval time.Duration, log interfaces.ILogger) (*PeerValidator, error) {
	registry, err := vr.NewValidatorregistry(registryAddress, backend)
	if err != nil {
		return nil, err
	}

	return &PeerValidator{
		registryAddr: registryAddress,
		walletAddr:   walletAddr,
		interval:     interval,
		registry:     registry,
		log:          log.Named("PEER_VAL"),
	}, nil
}

func (v *PeerValidator) Run(ctx context.Context) error {
	v.log.Info("initializing peer validator")
	validator, err := v.getValidator(ctx, v.walletAddr)
	if err != nil {
		v.log.Warnf("validator not registered, stopping peer validator")
		return nil
	} else {
		v.log.Info("validator record %+v", validator)
	}

	v.log.Infof("starting peer validator loop with interval %s", v.interval)
	ticker := time.NewTicker(v.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := v.validateRandomPeer(ctx)
			if err != nil {
				v.log.Error("failed to validate peer", err)
			}
		}
	}
}

func (v *PeerValidator) validateRandomPeer(ctx context.Context) error {
	validator, err := v.getRandomValidator(ctx)
	if err != nil {
		return err
	}
	v.log.Infof("validating peer %s, host %s", validator.Addr.Hex(), validator.Host)

	// validate peer
	ok, err := v.validatePeer(ctx, validator.Host)
	if err != nil {
		return err
	}
	if ok {
		v.log.Infof("peer %s is valid", validator.Addr.Hex())
		return nil
	}

	v.log.Infof("peer %s is invalid", validator.Addr.Hex())
	return v.complain(ctx, validator.Addr)
}

func (v *PeerValidator) getRandomValidator(ctx context.Context) (*vr.ValidatorRegistryValidator, error) {
	activeValidatorsCount, err := v.getActiveValidatorsCount(ctx)
	if err != nil {
		return nil, err
	}

	// get random from range 0 to activeValidatorsCount
	index := rand.Intn(int(activeValidatorsCount.Int64()))

	addr, err := v.getActiveValidatorAddr(ctx, big.NewInt(int64(index)))
	if err != nil {
		return nil, err
	}

	return v.getValidator(ctx, addr)
}

func (v *PeerValidator) getActiveValidatorsCount(ctx context.Context) (*big.Int, error) {
	// get active validators count from registry
	return v.registry.ActiveValidatorsLength(&bind.CallOpts{Context: ctx})
}

func (v *PeerValidator) getActiveValidatorAddr(ctx context.Context, index *big.Int) (common.Address, error) {
	addr, err := v.registry.GetActiveValidators(&bind.CallOpts{Context: ctx}, index, 1)
	if err != nil {
		return common.Address{}, err
	}
	if len(addr) == 0 {
		return common.Address{}, fmt.Errorf("no active validator at index %d", index)
	}
	return addr[0], nil
}

func (v *PeerValidator) getValidator(ctx context.Context, addr common.Address) (*vr.ValidatorRegistryValidator, error) {
	validator, err := v.registry.GetValidator(&bind.CallOpts{Context: ctx}, addr)
	return &validator, err
}

func (v *PeerValidator) complain(ctx context.Context, addr common.Address) error {
	_, err := v.registry.ValidatorComplain(&bind.TransactOpts{Context: ctx}, addr)
	return err
}

func (v *PeerValidator) validatePeer(ctx context.Context, host string) (bool, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	d := net.Dialer{}
	_, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		// server error
		return false, nil

		// client error
		// return false, err
	}
	return true, nil
}
