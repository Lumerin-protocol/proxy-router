package interfaces

import "gitlab.com/TitanInd/hashrouter/interop"

type IRoutableStreamsService interface {
	GetById(id string) (IRoutableStreamModel, error)
	TrySaveUniqueDestination(destUrl string) (IRoutableStreamModel, error)
	ChangeDestAll(dest interop.Dest) error
}
