package dataaccess

type DataAccess interface {
	GetContractsIDs() ([]string, error)
	GetContract(contractID string) (interface{}, error)
	CloseContract(contractID string, meta interface{}) error

	OnNewContract(func(contractID string))
	OnContractUpdated(func(contractID string))
	OnContractClosed(func(contractID string))
}
