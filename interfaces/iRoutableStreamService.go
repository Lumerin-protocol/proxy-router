package interfaces

type IRoutableStreamService interface {
	GetById(id string) (IRoutableStreamModel, error)
	TrySaveUniqueDestination(destUrl string) (IRoutableStreamModel, error)
}
