package message

import (
	"encoding/json"
	"errors"
	"fmt"

	"gitlab.com/TitanInd/hashrouter/lib"
)

var (
	ErrStratumV1Unmarshal = errors.New("cannot unmarshal stratumv1 message")
	ErrStratumV1Unknown   = errors.New("unknown stratumv1 message")
)

func ParseMessageToPool(raw []byte) (MiningMessageToPool, error) {
	message := &MiningGeneric{}
	err := json.Unmarshal(raw, message)
	if err != nil {
		return nil, lib.WrapError(ErrStratumV1Unmarshal, err)
	}

	switch message.Method {
	case MethodMiningSubscribe:
		return ParseMiningSubscribe(raw)

	case MethodMiningAuthorize:
		return ParseMiningAuthorize(raw)

	case MethodMiningSubmit:
		return ParseMiningSubmit(raw)

	default:
		return nil, lib.WrapError(fmt.Errorf("unknown message to pool: %s", raw), ErrStratumV1Unknown)
	}
}

func ParseMessageFromPool(raw []byte) (MiningMessageGeneric, error) {
	message := &MiningGeneric{}

	err := json.Unmarshal(raw, message)
	if err != nil {
		return nil, lib.WrapError(ErrStratumV1Unmarshal, err)
	}

	if message.Method == MethodMiningNotify {
		return ParseMiningNotify(raw)
	}
	if message.Method == MethodMiningSetDifficulty {
		return ParseMiningSetDifficulty(raw)
	}
	if message.Result != nil {
		return ParseMiningResult(raw)
	}

	return nil, lib.WrapError(fmt.Errorf("unknown message from pool: %s", raw), ErrStratumV1Unknown)
}
