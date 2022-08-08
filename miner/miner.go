package miner

// Miner is responsible for distributing the resources of a single miner across multiple destinations
// and falling back to default pool for unallocated resources
type Miner struct {
	minerModel MinerModel
	destSplit  DestSplit
}

// GetUnallocatedPercentage returns the percentage of power of a miner available to fulfill some contact
func (m *Miner) GetUnallocatedPercentage() uint8 {
	return m.destSplit.GetUnallocated()
}

// GetUnallocatedHashpower returns the available miner hashrate
// TODO: discuss with a team. As hashpower may fluctuate, define some kind of expected hashpower being
// the average hashpower value excluding the periods potential drop during reconnection
func (m *Miner) GetUnallocatedHashpower() int {
	// the remainder should be small enough to ignore
	return int(m.destSplit.GetUnallocated()) * m.minerModel.GetHashRate() / 100
}

// IsBusy returns true if miner is fulfilling at least one contract
func (m *Miner) IsBusy() bool {
	return m.destSplit.GetAllocated() > 0
}

// Allocate directs miner resources to the destination
func (m *Miner) Allocate(percentage uint8, destAddr, destUser, destPassword string) {
	m.destSplit.Allocate(percentage, destAddr, destUser, destPassword)
}

// Dellocate removes destination from miner's resource allocation
func (m *Miner) Dellocate(percentage uint8, destAddr, destUser, destPassword string) (ok bool) {
	return m.destSplit.Deallocate(destAddr, destUser)
}
