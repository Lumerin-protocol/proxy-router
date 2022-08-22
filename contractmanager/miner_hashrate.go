package contractmanager

import "gitlab.com/TitanInd/hashrouter/interfaces"

type HashrateListItem struct {
	Hashrate      uint64
	TotalHashrate uint64
	MinerID       string
}

func (m HashrateListItem) GetHashrate() uint64 {
	return m.Hashrate
}

func (m HashrateListItem) SetHashrate(hashrate uint64) {
	m.Hashrate = hashrate
}

func (m HashrateListItem) GetTotalHashrate() uint64 {
	return m.TotalHashrate

}

func (m HashrateListItem) GetSourceID() string {
	return m.MinerID
}

func (m HashrateListItem) GetPercentage() float64 {
	return float64(m.Hashrate) / float64(m.TotalHashrate)
}

type HashrateList []interfaces.IRoutableStreamFullfillment

func (m HashrateList) Len() int      { return len(m) }
func (m HashrateList) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m HashrateList) Less(i, j int) bool {
	return m[i].(*HashrateListItem).Hashrate < m[j].(*HashrateListItem).Hashrate
}
