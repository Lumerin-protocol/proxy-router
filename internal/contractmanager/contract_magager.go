package contractmanager

import "context"

type ContractManager interface {
	Run(ctx context.Context) error
}
