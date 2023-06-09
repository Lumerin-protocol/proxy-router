package allocator

import "context"

type Allocator interface {
	Run(ctx context.Context) error
	Allocate(ID string, hashrate float64, dest string, counter func(diff float64)) error
}
