package stratumv1_message

import (
	"encoding/json"
	"errors"

	"github.com/Lumerin-protocol/proxy-router/internal/lib"
	"github.com/Lumerin-protocol/proxy-router/internal/resources/hashrate/proxy/interfaces"
)

var (
	ErrStratumV1Unmarshal = errors.New("cannot unmarshal stratumv1 message")
	ErrStratumV1Unknown   = errors.New("unknown stratumv1 message")
)

func ParseStratumMessage(raw []byte) (interfaces.MiningMessageGeneric, error) {
	msg := &MiningGeneric{}
	err := json.Unmarshal(raw, msg)
	if err != nil {
		return nil, lib.WrapError(ErrStratumV1Unmarshal, err)
	}

	switch msg.Method {

	// client messages
	case MethodMiningSubscribe:
		return ParseMiningSubscribe(raw)

	case MethodMiningAuthorize:
		return ParseMiningAuthorize(raw)

	case MethodMiningSubmit:
		return ParseMiningSubmit(raw)

	case MethodMiningMultiVersion:
		return ParseMiningMultiVersion(raw)

	case MethodMiningConfigure:
		return ParseMiningConfigure(raw)

	// server messages
	case MethodMiningNotify:
		return ParseMiningNotify(raw)

	case MethodMiningSetDifficulty:
		return ParseMiningSetDifficulty(raw)

	case MethodMiningSetVersionMask:
		return ParseMiningSetVersionMask(raw)

	case MethodMiningSetExtranonce:
		return ParseMiningSetExtranonce(raw)

	case MethodExtranonceSubscribe:
		return ParseMiningExtranonceSubscribe(raw)

	default:
		if msg.Result != nil {
			return ParseMiningResult(raw)
		}

		return msg, nil
	}
}
