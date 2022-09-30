package protocol

import (
	"context"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/protocol/stratumv1_message"
)

type Hashrate interface {
	GetHashrate5minAvgGHS() int
	GetHashrate30minAvgGHS() int
	GetHashrate1hAvgGHS() int
}

type StratumV1SourceConn interface {
	GetID() string
	Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error)
	Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error
	GetWorkerName() string
	GetConnectedAt() time.Time
}

type StratumV1DestConn interface {
	ResendRelevantNotifications(ctx context.Context)
	SendPoolRequestWait(msg stratumv1_message.MiningMessageToPool) (*stratumv1_message.MiningResult, error)
	RegisterResultHandler(msgID int, handler StratumV1ResultHandler)
	SetDest(dest interfaces.IDestination, configure *stratumv1_message.MiningConfigure) error
	GetDest() interfaces.IDestination
	Read(ctx context.Context) (stratumv1_message.MiningMessageGeneric, error)
	Write(ctx context.Context, msg stratumv1_message.MiningMessageGeneric) error
	GetExtranonce() (string, int)
}

type StratumV1ResultHandler = func(a stratumv1_message.MiningResult) stratumv1_message.MiningMessageGeneric

type OnSubmitHandler = func(diff uint64, dest interfaces.IDestination)
type ListenerHandle int
