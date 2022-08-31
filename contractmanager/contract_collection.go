package contractmanager

import (
	"gitlab.com/TitanInd/hashrouter/data"
)

func NewContractCollection() *data.Collection[IContractModel] {
	return data.NewCollection[IContractModel]()
}
