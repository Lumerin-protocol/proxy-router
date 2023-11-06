package allocator

type MinerStatus uint8

const (
	MinerStatusVetting     MinerStatus = iota // vetting period
	MinerStatusFree                           // serving default pool
	MinerStatusBusy                           // fully or partially serving contract(s)
	MinerStatusPartialBusy                    // partially serving contract(s)
)

func (m MinerStatus) String() string {
	switch m {
	case MinerStatusVetting:
		return "vetting"
	case MinerStatusFree:
		return "free"
	case MinerStatusBusy:
		return "busy"
	case MinerStatusPartialBusy:
		return "partial_busy"
	}
	// shouldn't reach here
	return "ERROR"
}
