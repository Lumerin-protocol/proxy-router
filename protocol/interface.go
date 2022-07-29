package protocol

import (
	"context"

	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

type StratumV1SourceConn interface {
	GetID() string
	Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error)
	Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error
}

type StratumV1DestConn interface {
	SetDest(addr string, user string, pwd string) error
	Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error)
	Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error
	GetExtranonce() (string, int)
}

type StratumV1ResultHandler = func(a stratumv1_message.MiningResult) stratumv1_message.MiningMessageGeneric
