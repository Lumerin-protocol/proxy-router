package protocol

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

type StratumV1SourceConn interface {
	GetID() string
	Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error)
	Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error
	GetWorkerName() string
	GetConnectedAt() time.Time
}

type StratumV1DestConn interface {
	ResendRelevantNotifications(ctx context.Context) error
	SendPoolRequestWait(msg stratumv1_message.MiningMessageToPool) (*stratumv1_message.MiningResult, error)
	RegisterResultHandler(msgID int, handler StratumV1ResultHandler)
	SetDest(ctx context.Context, dest interfaces.IDestination, configure *stratumv1_message.MiningConfigure) error
	GetDest() interfaces.IDestination
	Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error)
	Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error
	GetExtranonce() (string, int)
	RangeConn(f func(key any, value any) bool)
	Close() error
}

type StratumV1ResultHandler = func(a stratumv1_message.MiningResult) stratumv1_message.MiningMessageGeneric
