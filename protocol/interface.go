package protocol

import "context"

type Connection interface {
	ChangePool(addr string) error
	WriteToPool(ctx context.Context, bytes []byte) error
	WriteToMiner(ctx context.Context, bytes []byte) error
}
