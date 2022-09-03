package protocol

import (
	"context"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

type StratumV1SourceConn interface {
	GetID() string
	Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error)
	Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error
	GetWorkerName() string
}

type StratumV1DestConn interface {
	SetDest(dest interfaces.IDestination) error
	GetDest() interfaces.IDestination
	Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error)
	Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error
	GetExtranonce() (string, int)
}

type StratumV1ResultHandler = func(a stratumv1_message.MiningResult) stratumv1_message.MiningMessageGeneric

type OnSubmitHandler = func(diff uint64, dest interfaces.IDestination)

type OnAuthorizeHandler = func(workername string, password string) error

type ListenerHandle int
