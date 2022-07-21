package miner

import "sync"

type MinerRepo struct {
	miners sync.Map
}

func NewMinerRepo() *MinerRepo {
	return &MinerRepo{
		miners: sync.Map{},
	}
}

func (p *MinerRepo) Load(ID string) (miner Miner, ok bool) {
	if val, ok := p.miners.Load(ID); ok {
		return val.(Miner), true
	}
	return nil, false
}

func (p *MinerRepo) Range(f func(miner Miner) bool) {
	p.miners.Range(func(key, value any) bool {
		miner := value.(Miner)
		return f(miner)
	})
}

func (p *MinerRepo) Store(miner Miner) {
	p.miners.Store(miner.GetID(), miner)
}

func (p *MinerRepo) Delete(id string) {
	p.miners.Delete(id)
}
