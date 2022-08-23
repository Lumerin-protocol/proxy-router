package protocol

import (
	"context"

	"gitlab.com/TitanInd/hashrouter/interop"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

type StratumV1SourceConn interface {
	GetID() string
	Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error)
	Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error
}

type StratumV1DestConn interface {
	SetDest(dest interop.Dest) error
	GetDest() interop.Dest
	Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error)
	Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error
	GetExtranonce() (string, int)
}

type StratumV1ResultHandler = func(a stratumv1_message.MiningResult) stratumv1_message.MiningMessageGeneric

type OnSubmitHandler = func(diff uint64, dest interop.Dest)
type ListenerHandle int
