package contractmanager

type HashrateListItem struct {
	Hashrate      uint64
	TotalHashrate uint64
	MinerID       string
}

func (m HashrateListItem) GetPercentage() float64 {
	return float64(m.Hashrate) / float64(m.TotalHashrate)
}

type HashrateList []HashrateListItem

func (m HashrateList) Len() int           { return len(m) }
func (m HashrateList) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m HashrateList) Less(i, j int) bool { return m[i].Hashrate < m[j].Hashrate }
