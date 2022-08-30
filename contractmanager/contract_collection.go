package contractmanager

import (
	"gitlab.com/TitanInd/hashrouter/data"
)

func NewContractCollection() *data.Collection[*Contract] {
	return data.NewCollection[*Contract]()
}
