package interfaces

type IRoutableStreamsService interface {
	GetById(id string) (IRoutableStreamModel, error)
	TrySaveUniqueDestination(destUrl string) (IRoutableStreamModel, error)
	ChangeDestAll(addr string, username string, pwd string) error
}
