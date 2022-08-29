package contractmanager

import (
	"sync"
)

type ContractCollection struct {
	items sync.Map
}

func NewContractCollection() *ContractCollection {
	return &ContractCollection{
		items: sync.Map{},
	}
}

func (p *ContractCollection) Load(ID string) (item *Contract, ok bool) {
	if val, ok := p.items.Load(ID); ok {
		return val.(*Contract), true
	}
	return nil, false
}

func (p *ContractCollection) Range(f func(item *Contract) bool) {
	p.items.Range(func(key, value any) bool {
		item := value.(*Contract)
		return f(item)
	})
}

func (p *ContractCollection) Store(item *Contract) {
	p.items.Store(item.GetID(), item)
}

func (p *ContractCollection) Delete(ID string) {
	p.items.Delete(ID)
}
